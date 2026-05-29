# devflow-skills CLI 分发工具 — 方案设计

## 1. 选型决策

### 方案对比

| 维度 | 方案 A：npm 薄层 + Go 二进制 | 方案 B：纯 Go（go install） | 方案 C：纯 npm 包 |
|------|----------------------------|---------------------------|------------------|
| 用户覆盖 | 极好（npx 无安装门槛） | 好（需 Go 环境） | 最好（Node.js 普遍安装） |
| 离线可用 | 是（二进制内嵌技能） | 是 | 否（需 npm 下载技能文件） |
| 二进制体积 | ~3-5MB（含嵌入技能） | ~3-5MB | N/A |
| 跨平台 | Go 交叉编译 | Go 交叉编译 | N/A |
| 发布复杂度 | 中等（GitHub Release + npm publish） | 低（仅 GitHub Release + tag） | 高（管理 80 个技能文件的 npm 包） |
| 版本管理 | GitHub Release 版本号 | Go module version | npm version |

### 决策

采用 **方案 A + B 组合**：Go 二进制为主（内嵌技能，离线可用），npm 为分发渠道（薄层 wrapper，自动下载正确平台的二进制）。

理由：
1. `npx devflow-skills` 覆盖最广的场景，用户无需预装 Go
2. Go 二进制离线可用，不依赖网络下载技能文件
3. 单一二进制文件，部署简单
4. 同时保留 `go install` 路径给 Go 开发者

## 2. 架构设计

```
┌──────────────────────────────────────┐
│           npx devflow-skills         │
│  (npm 薄层 wrapper: install.js)      │
│                                      │
│  1. 检查本地是否有 Go 二进制         │
│  2. 没有 → 从 GitHub Releases 下载   │
│  3. 转发所有参数给 Go 二进制         │
└──────────────┬───────────────────────┘
               │
┌──────────────▼───────────────────────┐
│      Go 二进制 (devflow-skills)       │
│                                      │
│  ┌──────────────────────────────┐    │
│  │  embed.FS: plugins/          │    │
│  │  (80 个 SKILL.md 编译进二进制) │    │
│  └──────────────────────────────┘    │
│                                      │
│  ┌──────────────────────────────┐    │
│  │  CLI 命令分发                 │    │
│  │  install / list / uninstall  │    │
│  └──────────────────────────────┘    │
│                                      │
│  ┌──────────────────────────────┐    │
│  │  平台适配器                   │    │
│  │  opencode / claude / codex   │    │
│  └──────────────────────────────┘    │
└──────────────────────────────────────┘
```

## 3. 目录结构

```
devflow-skills/
├── main.go                        # Go 入口 (package main)
│                                  # //go:embed plugins
├── go.mod                         # module github.com/zhouhao4221/devflow-skills
├── go.sum
├── Makefile                       # 构建脚本
├── package.json                   # npm 包定义
├── install.js                     # npm bin → 下载并执行 Go 二进制
├── plugins/                       # 技能源文件（80 个 SKILL.md）
│   ├── req/skills/               # 46 个
│   ├── api/skills/               # 8 个
│   ├── pm/skills/               # 14 个
│   ├── diag/skills/             # 5 个
│   └── uat/skills/             # 7 个
├── scripts/                       # 原有 shell 脚本（保留）
├── skill-bindings.json            # 原有配置
├── docs/
│   ├── requirements/
│   │   └── cli-distribution.md    # 需求文档
│   └── design/
│       └── cli-distribution.md    # 本文件
├── .gitignore
└── README.md
```

## 4. Go 代码设计

### 核心数据流

```
CLI args → flag.Parse → 路由到子命令
                          │
              ┌───────────┼───────────┐
              ▼           ▼           ▼
           install      list      uninstall
              │           │           │
              ▼           ▼           ▼
         read embed.FS  read embed.FS  os.Remove
              │           │
              ▼           ▼
         平台适配器       fmt.Print
              │
    ┌─────────┼─────────┐
    ▼         ▼         ▼
 opencode   claude    codex
 (扁平)    (分层)    (扁平+yaml)
```

### 技能查找逻辑

用户可通过两种方式指定技能：
1. **扁平名**：`req-dev` → 拆分为 plugin=req, name=dev
2. **原名**：`dev` → 在所有插件中搜索，找到唯一匹配则使用，多个匹配则报错提示

### 平台适配器接口

```go
type Platform interface {
    // 安装单个技能到 targetDir
    Install(targetDir string, plugin string, name string, content []byte) error
    // 获取技能目标路径（用于判断是否已安装）
    TargetPath(targetDir string, plugin string, name string) string
}
```

## 5. npm 薄层包装设计

`install.js` 逻辑：

```
1. 解析 package.json 中的 version → 确定 Go 二进制版本
2. 检测当前平台和架构 (os.arch, os.platform)
3. 检查本地缓存：~/.devflow-skills/<version>/devflow-skills[.exe]
4. 如果不存在 → 下载 https://github.com/zhouhao4221/devflow-skills/releases/download/v<version>/devflow-skills_<os>_<arch>[.exe]
5. 转发 child_process.spawn(二进制路径, process.argv.slice(2))
```

`package.json` 结构：

```json
{
  "name": "devflow-skills",
  "version": "0.1.0",
  "description": "一键安装/管理 devflow-skills AI 技能包",
  "bin": {
    "devflow-skills": "./install.js"
  },
  "files": ["install.js", "package.json"]
}
```

## 6. 发布流程

### 6.1 Go 二进制发布

```
1. git tag v0.1.0
2. git push --tags
3. CI (GitHub Actions) 触发构建：
   - 交叉编译: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
   - 创建 GitHub Release
   - 上传二进制到 Release Assets
```

### 6.2 npm 包发布

```
1. 更新 package.json version
2. npm publish (发布 install.js + package.json，不含二进制)
3. 用户执行 npx devflow-skills → install.js 按需下载 Go 二进制
```

### 6.3 版本策略

- Go module 版本号与 git tag 保持一致
- npm 版本号与 Go 版本号保持一致
- 技能内容更新 → 发新版本（所有包含嵌入技能的文件更新）

## 7. 兼容性

- 保留现有 `scripts/setup-*.sh` 脚本，不删除
- CLI 工具生成与现有 shell 脚本相同格式的输出
- skill-bindings.json 不受影响
