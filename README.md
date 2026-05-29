# DevFlow Skills

DevFlow 插件技能集合 —— 独立技能包，包含 [DevFlow](https://github.com/zhouhao4221/devflow-claude) 所有插件的 AI 执行指令（SKILL.md）。支持 OpenCode、Claude Code、Codex 三大 AI 工具。

## 快速开始

使用 CLI 工具一键安装技能到目标 AI 工具目录：

```bash
# 安装全部技能到 OpenCode
npx devflow-skills install --tool opencode --all

# 安装指定技能
npx devflow-skills install --tool opencode --skill req-dev --skill req-review

# 列出所有可用技能
npx devflow-skills list

# 卸载技能
npx devflow-skills uninstall --tool opencode --all
```

首次运行 `npx` 时会自动下载对应平台 Go 二进制，也可通过 Go install 安装：

```bash
go install github.com/zhouhao4221/devflow-skills@latest
devflow-skills install --tool claude --all
```

## 概述

本仓库从 [devflow-claude](https://github.com/zhouhao4221/devflow-claude) 主仓库中提取所有 SKILL.md 文件，作为独立的技能包提供给 CI 校验、版本管理和跨工具分发。每个 SKILL.md 文件定义了 AI 工具在执行对应命令时的行为指引。

### CLI 命令速览

| 命令 | 说明 |
|------|------|
| `install` | 安装技能到目标 AI 工具目录 |
| `list` | 列出所有可用技能，支持按插件过滤和 JSON 输出 |
| `uninstall` | 从目标 AI 工具目录卸载技能 |

| 参数 | 说明 |
|------|------|
| `--tool opencode\|claude\|codex` | 目标 AI 工具（必填） |
| `--skill <NAME>` | 要操作的技能名，支持重复指定多个 |
| `--all` | 操作所有技能（与 `--skill` 互斥） |
| `--dir <PATH>` | 目标项目根目录（默认 `.`） |
| `--plugin <NAME>` | 按插件过滤（`list` 命令） |
| `--format text\|json` | 输出格式（`list` 命令） |

### CLI 安装示例

```bash
# OpenCode: 安装到 .agents/skills/（扁平命名，如 req-dev）
npx devflow-skills install --tool opencode --skill req-dev

# Claude Code: 安装到 plugins/req/skills/dev/（分层目录）
npx devflow-skills install --tool claude --all

# Codex: 安装到 .agents/skills/（扁平命名，同上）
npx devflow-skills install --tool codex --all

# 指定目标目录
npx devflow-skills install --tool opencode --all --dir ../my-project
```

## 仓库结构

```
plugins/
├── req/skills/     # 需求管理技能（46 个）
├── api/skills/     # API 对接技能（8 个）
├── pm/skills/      # 项目管理技能（14 个）
├── diag/skills/    # 生产诊断技能（5 个）
└── uat/skills/     # 验收测试技能（7 个）
skill-bindings.json  # 命令-技能映射关系
```

## 技能分类

### req — 需求全流程管理
覆盖从需求分析、评审、开发、测试到归档的完整生命周期，包括智能开发（`dev`）、规范提交（`commit`）、版本发布（`release`）、代码审查（`review-pr`）、自然语言调度（`natural-language-dispatcher`）等 46 个技能。

**自动触发技能**：
| 技能 | 触发场景 |
|------|---------|
| `requirement-analyzer` | 创建/编辑需求时自动分析 |
| `dev-guide` | 开发阶段按分层架构引导 |
| `quick-fix-guide` | 快速修复时引导 |
| `test-guide` | 测试阶段引导回归和新建 |
| `prd-analyzer` | PRD 编辑时辅助完善 |
| `code-impact-analyzer` | 需求变更时分析代码影响 |
| `changelog-generator` | 自动生成版本变更说明 |
| `version-bumper` | 发版时自动推导版本号 |

### api — API 对接
支持 Swagger/OpenAPI 解析、字段映射、代码生成、接口搜索等 8 个技能。

### pm — 项目管理助手
从 PRD、需求文档和 Git 记录中提取数据，生成汇报、统计、方案等 14 个技能。

### diag — 生产诊断
只读诊断 + 修复建议，包含审计、堆栈分析等 5 个技能，含安全风控 Hook。

### uat — 用户验收测试
UI 验收测试工作流，含测试执行器、报告生成等 7 个技能。

## skill-bindings.json

`skill-bindings.json` 声明了命令与技能之间的映射关系：

```json
{
  "plugins": {
    "req": {
      "commands": {
        "dev": {
          "primarySkill": "dev",
          "additionalSkills": ["dev-guide"]
        }
      }
    }
  },
  "allSkills": {
    "req": ["dev", "dev-guide", ...]
  }
}
```

- `commands.<name>.primarySkill`: 命令直接对应的技能
- `commands.<name>.additionalSkills`: 命令执行过程中自动触发的辅助技能
- `allSkills`: 各插件拥有的全部技能清单

CI 会验证：
1. `allSkills` 中声明的技能都有对应的 `SKILL.md` 文件
2. 实际存在的 `SKILL.md` 文件都在 `allSkills` 中有声明
3. 命令中引用的技能名都在 `allSkills` 中存在

## CI 校验

每次 push 或 PR 时自动运行 CI（`.github/workflows/ci.yml`），包含：

| 检查项 | 说明 |
|--------|------|
| validate-source | 全量校验 80 个 SKILL.md 是否符合 agentskills.io 规范（name 正则、description 长度） |
| validate-claude | setup-claude 脚本输出完整性检查 |
| validate-opencode | setup-opencode 输出 frontmatter 格式 + 计数 |
| validate-codex | setup-codex + openai.yaml 输出完整性 |

本地手动运行：`./scripts/validate-skills.sh --ci`

---

## 手动安装（适配器脚本）

CLI 工具（上节）已覆盖所有常见场景。如需在不同环境分配不同路径、或嵌入 CI 流程，可通过适配器脚本手动生成。

devflow-skills 作为唯一起源，通过适配器脚本生成各工具兼容的技能目录。以下分别说明 Claude Code、OpenCode、Codex 三工具的安装方式。

### 前置条件

- 已 clone 本仓库
- Python 3 可用（用于 `_get-description.py`）

### Claude Code

Claude Code 通过插件 marketplace 加载技能，使用分层目录结构（`plugins/<plugin>/skills/<name>/SKILL.md`）。

```bash
# 同步技能到 devflow-claude 仓库
./scripts/setup-claude.sh . ../devflow-claude
# 参数: <devflow-skills路径> [devflow-claude路径]
```

产物路径：`../devflow-claude/plugins/<plugin>/skills/<name>/SKILL.md`（80 个文件，与源结构一致）。由 devflow-claude 的 `plugin.json` 加载。

### OpenCode

OpenCode 从 `.agents/skills/` 发现技能，使用扁平化命名（`<plugin>-<name>`）。

```bash
# 生成 OpenCode 兼容的技能目录
./scripts/setup-opencode.sh . ~/.agents/skills
# 参数: <devflow-skills路径> [输出路径，默认 .agents/skills/]
```

产物路径：`~/.agents/skills/<plugin>-<name>/SKILL.md`（80 个目录，name 已自动更新为扁平化前缀）。OpenCode 启动时自动发现所有技能。

### Codex

Codex 同样从 `.agents/skills/` 发现技能，兼容 agentskills.io 规范。

```bash
# 基础安装（仅技能目录）
./scripts/setup-codex.sh . ~/.agents/skills

# 附加生成 UI 元数据清单（agents/openai.yaml）
./scripts/setup-codex.sh . ~/.agents/skills --gen-openai-yaml
```

产物路径：`~/.agents/skills/<plugin>-<name>/SKILL.md`（80 个目录）。`--gen-openai-yaml` 额外生成 `agents/openai.yaml`，提供 display_name、description 等 UI 元数据。

### 验证安装

安装完成后运行校验脚本确认一切正常：

```bash
./scripts/validate-skills.sh --ci
```

通过后输出 `SUCCESS: all validations passed`，共 8 项检查全部 PASS。

---

## 设计理念

### 为什么做跨工具 Skill 通用化

当前 80 个 SKILL.md 以 Claude Code marketplace 为原生格式（`plugins/<plugin>/skills/<name>/SKILL.md`），在 Codex、OpenCode 等工具中无法直接加载。各工具的 skill 发现机制不同：

| 工具 | 发现路径 | 命名方式 |
|------|---------|----------|
| Claude Code | `plugins/<plugin>/skills/<name>/SKILL.md` | 插件层作用域 |
| OpenCode | `.agents/skills/<name>/SKILL.md` | 全局扁平 |
| Codex | `.agents/skills/<name>/SKILL.md` | 全局扁平 |

**核心思路**：devflow-skills 作为唯一起源（single source of truth），不修改 80 个 SKILL.md 正文内容，通过适配层脚本（adapter scripts）为各工具生成各自的兼容产物。

### 关键设计决策

#### D1：统一标准 — 采用 agentskills.io 规范

[agentskills.io](https://agentskills.io/specification) 是 Open Agent Skills Standard，Codex 原生内置、OpenCode 兼容支持。格式要求为 YAML frontmatter（`name` + `description` 必填）+ Markdown 正文。当前 80 个 SKILL.md 全部使用此格式，零修改即兼容核心要求。

选择理由：
- 已被业界采纳（Codex 原生、OpenCode 兼容），非私有标准
- 格式简洁（两个必填字段 + 可选扩展），与现有 SKILL.md 完全兼容
- 有明确的命名正则约束（`^[a-z0-9]+(-[a-z0-9]+)*$`），保证跨工具一致性

#### D2：命名策略 — `<plugin>-<name>` 扁平化

Claude Code 按插件分层（`req/dev`、`api/search`），名字无需全局唯一。但 OpenCode/Codex 是全局扁平名字空间。采用 `<plugin>-<name>` 前缀方案（如 `req-dev`、`api-search`）：

- **全局唯一**：多个插件下的重名技能（如 `help`、`init`、`new`）加上前缀后自然去重（`req-help` vs `api-help` vs `pm-help`）
- **可读性强**：一眼可知技能所属插件
- **符合 agentskills.io 正则**：全部由小写字母、数字、连字符组成

命名变更在适配层处理（复制时通过 `sed` 更新 frontmatter `name` 字段），源头 SKILL.md 正文不动。

#### D3：适配路径 — `.agents/skills/` 作为三工具最大公约数

OpenCode 和 Codex 均原生扫描 `.agents/skills/`，且 OpenCode 也兼容 `.claude/skills/`。选择 `.agents/skills/` 作为适配器统一输出路径，确保：
- OpenCode 和 Codex 零配置可用
- Claude Code 仍通过 `plugins/*/skills/` 加载（setup-claude.sh 直接 copy，不使用 `.agents/skills/`）

#### D4：不修改源头 SKILL.md

80 个 SKILL.md 始终保持原样，所有跨工具适配（name 扁平化、路径重组）都在适配器脚本的复制过程中完成。好处：
- 源头文件干净、可独立维护
- 各工具适配逻辑独立，互不干扰
- CI 校验只需对源头文件生效

### 架构总览

```
                         ┌────────────────────┐
                         │   devflow-skills    │  ← 唯一起源
                         │  plugins/<plugin>/  │     80 个 SKILL.md
                         │  skills/<name>/     │     零修改
                         │  SKILL.md           │
                         │  skill-bindings.json│
                         └────────┬───────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
           ┌───────────┐  ┌───────────┐  ┌───────────┐
           │  Claude   │  │ OpenCode  │  │   Codex   │
           │  Adapter  │  │  Adapter  │  │  Adapter  │
           └─────┬─────┘  └─────┬─────┘  └─────┬─────┘
                 ▼              ▼              ▼
        plugins/*/skills/  .agents/skills/  .agents/skills/
        (分层插件结构)      (扁平前缀命名)    (扁平前缀命名)
```

---

## 许可证

[Apache License 2.0](LICENSE)
