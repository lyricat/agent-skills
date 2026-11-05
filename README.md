# Agent Skills

一组面向 Codex 的工程 Skill。每个 Skill 都是独立目录，核心说明位于各自的 `SKILL.md`。

## 可用 Skill

| Skill | 用途 |
| --- | --- |
| [`review-code`](review-code/SKILL.md) | 审查工作区、提交、分支、补丁或 PR 中的代码变更，识别具体缺陷、代码安全风险、兼容性问题和关键测试缺口。 |
| [`simplify-code`](simplify-code/SKILL.md) | 在保留行为、公开契约和安全边界的前提下，删除不必要的抽象、分支、状态、间接层、配置和依赖。 |
| [`factory`](factory/SKILL.md) | 用 `factory_board` 分支上 `.factory/` 目录中的 Markdown 文件管理功能和问题，由编码代理完成构建、审查和合并。 |

### `review-code`

- 只报告能够由代码、测试或权威资料验证的问题。
- 每条 finding 必须给出具体的触发条件和错误结果示例。
- 安全审查只覆盖变更涉及的代码和构建逻辑，不检查 DNS 等外部基础设施。
- 依赖变更只检查具体的构建、API 和运行时兼容性回归；不做供应链安全审计、依赖漏洞扫描或安全公告查询。
- 默认只审查，不修改文件、提交代码、推送分支或发布评论。

安全检查细则见 [`security-review.md`](review-code/references/security-review.md)。

### `simplify-code`

- 先确定必须保留的行为、接口、数据和安全属性。
- 删除没有实际职责的薄包装、单实现抽象、重复状态、重复规则和无依据的扩展点。
- 不以减少行数为目标，也不把复杂性转移到更隐蔽的表达式或更宽泛的抽象中。
- 修改正式代码逻辑前，先用测试固定现有行为；修改后运行相关检查。

### `factory`

- 每个条目是一个 Markdown 文件，所在的目录就是它的进度：`new/` → `review/` → `ready/` → `done/`。
- build 构建 `new/` 中的一个条目并移入 `review/`；review 审查 `review/` 中的一个条目，没有 P0 或 P1 问题时移入 `ready/`；ship 在所有者确认后把 `ready/` 中的条目合并进目标分支并移入 `done/`。
- 目标分支由 `.factory/factory.toml` 中的 `base` 决定，默认是 `master`。构建从这个分支开始，ship 也合并进这个分支。

要让完成的条目合并进 `develop` 而不是 `master`，在该仓库看板的 `factory.toml` 中设置：

```toml
base = "develop"
```

`base` 对整个仓库生效，不能为单个条目另行指定。全部配置项见 [`factory.toml`](factory/references/factory.toml)。

## 安装

Codex 可以从 `~/.codex/skills` 加载个人 Skill。在本仓库根目录执行：

```bash
mkdir -p ~/.codex/skills
cp -R review-code simplify-code ~/.codex/skills/
```

只需要其中一个 Skill 时，仅复制对应目录。

## 使用

Codex 可以根据 Skill 的描述自动选择，也可以在提示词中显式调用：

```text
$review-code 审查当前工作区相对 master 的变更。

$simplify-code 在不改变行为的前提下简化 src/example.go。
```

更多格式与工作方式见 [Codex Skills 官方文档](https://developers.openai.com/plugins/concepts/skills)。
