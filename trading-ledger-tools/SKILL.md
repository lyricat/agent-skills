---
name: trading-ledger-tools
description: "使用 Skill 自带的 bal.sh、report.sh 和 monthly-returns.py 管理以 USDT 计价的 Beancount 交易账户账本：创建最小账本，记录资产、负债和成本批次，查询仓位与持仓成本，生成已实现盈亏、净资产和月度收益报告，并在变更后校验。适用于交易账户的仓位、均价、盈亏、注资与分配、收益曲线及账户命名；不适用于日常收支账本。"
---

# Trading Ledger Tools

使用本 Skill 的 `scripts/` 完成交易账户的建账、查询和报告，不重写其中的 BQL、估值或制图逻辑。这套工具以 USDT 为计价币。

## 选择工作方式

- 查询已有账本：直接使用下方命令，不读取建账模板。
- 创建或修改账本：先完整读取 [账户与记账规范](references/ledger-conventions.md)，再以 [最小账本模板](assets/minimal-ledger.bean) 为起点；不要把模板内容凭记忆重写。
- 只做查询时不要修改账本或重新生成文件。只有用户要求创建或更新曲线时才写入 SVG 和 CSV。

将当前加载的 `SKILL.md` 所在目录解析为 `SKILL_DIR`；不要假定 Skill 安装在某个固定路径。除模板与规范链接外，从目标仓库根目录运行命令，并始终显式传入 `--ledger TARGET/main.bean`。用户已指定账本时直接使用；只有上下文无法确定目标时才询问。先读取目标目录内适用的 `AGENTS.md`。

## 查询仓位

```bash
"$SKILL_DIR/scripts/bal.sh" --ledger TARGET/main.bean --plain
```

`--plain` 用于采集结果；只有用户要终端表格时才用 `--pretty`。报告所有非零资产时：

- 分别列出账户持有量、负债和净持仓。
- 持仓均价只取账本成本批次；没有成本批次时写“无法计算”，不得用市场价格代替。
- 多空成本数量完全相抵时，均价写“不适用”。
- USD、USDC、USDS、USDT 按 `1 USDT` 面值计算。

## 生成完整报告

```bash
"$SKILL_DIR/scripts/report.sh" --ledger TARGET/main.bean --price SYMBOL=USER_SUPPLIED_PRICE --plain
```

`--price` 可重复。为每个仍有非零资产账户或负债账户的非稳定币传入价格；即使同币种净持仓为零，也不能漏掉。优先使用用户给出的行情、成交价或明确指定的估值价，并说明口径。

未传入的非稳定币价格会从 Binance 公共行情接口读取。实际发起网络请求前遵守当前环境的联网与授权要求；不能联网时列出缺价币种，不得虚构价格。`SYMBOL=1` 只可用于不检查实际估值数值的回归测试。

解释结果时：

- “历史净已实现盈亏”是已实现交易盈亏减去已折算手续费，不等于累计经济收益。
- “累计注资”记录历史投入；“累计提取与利润分配”包含利润分配和退股提取。
- 存在 `!` 交易时，说明待核实笔数及其可能影响。
- 给出净资产时同时说明估值日期和价格来源。

## 生成月度曲线

```bash
python3 "$SKILL_DIR/scripts/monthly-returns.py" \
  --ledger TARGET/main.bean \
  --output TARGET/monthly-returns.svg \
  --csv TARGET/monthly-returns.csv \
  --as-of YYYY-MM-DD
```

始终显式指定截止日。默认模式从 Binance 读取所需非稳定币的历史日线收盘价。用户禁止联网，或账本已包含每个持仓月份的月末价格时，增加 `--offline`。离线模式报缺价时列明月份和币种，不得自行造价。空账本尚无可统计交易，不能生成曲线。

以 CSV 的 `cumulative_profit_usdt` 作为累计经济收益，不要把已实现盈亏重复加进去。曲线与即时报告有小额差异时，先核对估值价格和时点。

## 原始查询

普通请求使用公开脚本。只有用户需要原始 CSV 或排查查询逻辑时，才直接运行：

```bash
bean-query --format csv TARGET/main.bean < "$SKILL_DIR/scripts/realized.bql"
bean-query --format csv TARGET/main.bean < "$SKILL_DIR/scripts/ror.bql"
```

`$SKILL_DIR/scripts/terminal-table.sh` 是脚本库，不单独执行。

## 校验

创建或修改账本后至少运行：

```bash
bean-check TARGET/main.bean
"$SKILL_DIR/scripts/bal.sh" --ledger TARGET/main.bean --plain
```

需要估值时，再以真实价格运行 `report.sh`。若仓库提供 `scripts/tests/test-ledgers.sh`，也运行它。

修改脚本正式逻辑时，先用 `rg --files` 找到与 `bal`、`report`、`monthly-returns`、`realized` 或 `ror` 对应的现有测试；先更新并运行测试确认失败，再实现，最后运行相关测试和全账本校验。

失败时报告具体命令、退出码和第一条可处理错误；不要把缺价、待核实交易或无法计算的成本隐藏成成功结果。
