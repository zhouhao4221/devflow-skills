# devflow-skills CLI 分发工具 — 需求文档

## 1. 背景

Phase 1-3 完成了多 AI 工具适配（Claude Code / OpenCode / Codex），技能定义已抽取为平台无关的 `devflow-skills` 独立仓库（80 个 SKILL.md）。但当前用户仍需手动运行 shell 脚本来安装技能，缺乏统一、便捷的分发机制。

## 2. 目标

提供一个 CLI 工具 `devflow-skills`，用户在任何 AI 工具环境下都能一键安装/管理 devflow-skills，无需手动操作。

## 3. 用户故事

- 作为工程师，我希望能用 `npx devflow-skills install --tool opencode --all` 一键安装所有技能到我的 opencode 项目
- 作为工程师，我希望能按需安装特定技能：`npx devflow-skills install --tool opencode --skill req-dev`
- 作为工程师，我希望能查看所有可用技能及其描述
- 作为工程师，我希望能卸载已安装的技能

## 4. CLI 命令规范

### 4.1 `install` — 安装技能

```
devflow-skills install --tool <TOOL> [--skill <NAME>...] [--all] [--dir <PATH>]
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--tool` | 是 | 目标 AI 工具：`opencode` / `claude` / `codex` |
| `--skill` | 否 | 要安装的技能名（可重复指定多个）。格式为 `<plugin>-<name>`（扁平命名）或 `<name>`（技能原名） |
| `--all` | 否 | 安装所有技能。与 `--skill` 互斥 |
| `--dir` | 否 | 目标项目根目录，默认当前目录 |

**行为**：

| 工具 | 安装路径 | 命名规则 |
|------|---------|---------|
| `opencode` | `<dir>/.agents/skills/<plugin>-<name>/SKILL.md` | 扁平 `{plugin}-{name}`，更新 frontmatter 的 `name` 字段 |
| `codex` | `<dir>/.agents/skills/<plugin>-<name>/SKILL.md` | 同 opencode，扁平命名。可选生成 `agents/openai.yaml` |
| `claude` | `<dir>/plugins/<plugin>/skills/<name>/SKILL.md` | 保持原有分层结构，不修改 frontmatter |

**错误处理**：
- 目标目录不存在 → 提示并退出
- 指定技能名不存在 → 列出可用技能并退出
- `--skill` 与 `--all` 同时使用 → 提示互斥

**安装后的提示**：
```
已安装 3 个技能到 .agents/skills/：
  req-dev (需求开发 - 启动或继续开发)
  req-review (需求评审)
  pm-weekly (生成周报)
下一步：重启 AI 工具或刷新技能列表即可使用。
```

### 4.2 `list` — 列出可用技能

```
devflow-skills list [--plugin <NAME>] [--format text|json]
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--plugin` | 否 | 按插件过滤：`req` / `api` / `pm` / `diag` / `uat` |
| `--format` | 否 | 输出格式，默认 `text` |

**输出示例（text 格式）**：
```
req 插件 (46 个技能)：
  req                    需求管理 - 初始化/配置需求管理环境
  dev                    需求开发 - 启动或继续开发
  review                 需求评审 - 评审需求文档
  ...

api 插件 (8 个技能)：
  api                    API 对接 - 初始化 API 对接工作区
  ...
```

**输出示例（json 格式）**：
```json
{
  "plugins": {
    "req": [
      {"name": "req", "description": "需求管理 - 初始化/配置需求管理环境"},
      {"name": "dev", "description": "需求开发 - 启动或继续开发"}
    ]
  }
}
```

### 4.3 `uninstall` — 卸载技能

```
devflow-skills uninstall --tool <TOOL> [--skill <NAME>...] [--all] [--dir <PATH>]
```

参数与 `install` 相同。删除对应工具目录下的技能文件。

**行为**：
- 技能不存在 → 提示「技能未安装，无需卸载」
- `--all` → 删除该工具下所有 devflow 技能目录
- 卸载后提示已删除的技能列表

## 5. 分发方式

| 方式 | 命令 | 说明 |
|------|------|------|
| npm（推荐） | `npx devflow-skills install --tool opencode --all` | 薄层包装，自动下载对应平台的 Go 二进制 |
| go install | `go install github.com/zhouhao4221/devflow-skills@latest` | 直接安装 Go 二进制 |

## 6. 非功能需求

- Go 二进制不依赖外部运行时，单一可执行文件
- 技能内容通过 `embed.FS` 编译进二进制，离线可用
- 支持 macOS / Linux / Windows（通过交叉编译）
- npm 包体积 < 2KB（仅含 wrapper 脚本，实际二进制从 GitHub Releases 下载）
