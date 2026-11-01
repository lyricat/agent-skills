# 账户与记账规范

创建或修改兼容账本时使用本规范。只查询现有账本时无需读取。

## 兼容范围

这组脚本固定以 USDT 估值，并把 USD、USDC、USDS、USDT 视为按面值计算的稳定币。其他币种都需要 USDT 价格。

账户名使用冒号分层，每段以大写字母开头，不使用空格。名称表达经济用途，不把币种、订单号或临时状态塞进账户名。账户必须在首次记账前 `open`。

## 账户树

以下根账户和命名位置会影响脚本统计：

| 账户 | 用途 |
| --- | --- |
| `Assets:VENUE:Spot` | 交易所或托管方持有的现货资产；把 `VENUE` 换成稳定的场所名 |
| `Assets:Wallet:LABEL` | 自托管钱包；把 `LABEL` 换成稳定的钱包标签 |
| `Liabilities:Loans:LENDER` | 向外部主体借入的资产；借款不与资产余额抵销 |
| `Equity:OpeningBalances` | 启用账本前已有的净资产，只在期初使用 |
| `Equity:Contributions[:OWNER]` | 外部投入的累计本金；需要按出资人统计时增加子账户 |
| `Equity:Distributions[:OWNER]` | 利润分配、提款和退股返还；需要按收款人统计时增加子账户 |
| `Equity:Adjustments` | 有依据的非经营性账面调整，不用于填平未知差额 |
| `Income:Gains:Trade` | 已实现交易盈亏；必须位于 `Income:Gains` 之下 |
| `Expenses:VENUE:TradeFees` | 交易手续费；费用账户名必须以 `Fees` 结尾 |

`VENUE`、`LABEL`、`LENDER`、`OWNER` 是说明中的占位符，不应原样留在正式账本。模板中的 `Exchange` 可以在场所明确后改名。一个账户只承担一种经济用途。

## 资金与负债

注资记为资产增加、权益贷记：

```beancount
2026-01-01 * "Capital contribution"
  Assets:Exchange:Spot       10000 USDT
  Equity:Contributions     -10000 USDT
```

分配或退股记为资产减少、分配权益借记：

```beancount
2026-02-01 * "Distribution"
  Assets:Exchange:Spot      -1000 USDT
  Equity:Distributions       1000 USDT
```

`Contributions` 和 `Distributions` 是历史累计流量。退回本金通常仍记入 `Distributions`，不要冲销过去的 `Contributions`。需要说明本金与利润的组成时，用交易说明、元数据或 `Distributions` 子账户保留细节。

借入资产时同时增加持有资产和负债：

```beancount
2026-01-05 * "Borrow BTC"
  Assets:Exchange:Spot         1 BTC
  Liabilities:Loans:Lender    -1 BTC
```

还款使用相反方向。自有账户之间转账只移动 `Assets:`；不要产生收入、支出或权益。链上费另记到以 `Fees` 结尾的费用账户。

## 成本批次与交易

需要持仓均价的买入必须建立 `{...}` 成本批次：

```beancount
2026-01-10 * "BUY BTC"
  Assets:Exchange:Spot       0.1 BTC {60000 USDT}
  Assets:Exchange:Spot     -6000 USDT
```

卖出时同时写原成本和成交价，让 Beancount 计算已实现盈亏：

```beancount
2026-02-10 * "SELL BTC"
  Assets:Exchange:Spot      -0.1 BTC {60000 USDT} @ 70000 USDT
  Assets:Exchange:Spot      7000 USDT
  Income:Gains:Trade
```

`{60000 USDT}` 是单位成本，`@ 70000 USDT` 是本次成交价。只有 `@` 而没有 `{}` 不会建立可用于持仓均价的成本批次。多笔成交价不同时，逐笔写 posting，或在保留原始明细的前提下使用可核验的加权价格。

手续费与成交金额分开：

```beancount
2026-02-10 * "Trading fee"
  Assets:Exchange:Spot          -2 USDT
  Expenses:Exchange:TradeFees    2 USDT
```

手续费使用实际扣除币种。`Expenses:Fees:Trade` 不以 `Fees` 结尾，不会进入脚本的手续费汇总，不能采用。

## 价格、状态与来源

离线月度曲线要求每个存在非零非稳定币持仓的月份都有当月 USDT 价格：

```beancount
2026-01-31 price BTC 65000 USDT
```

市场价格只用于估值，不得代替成本批次。非稳定币注资、分配或费用也要有可用于折算的账本价格。

确认数据使用 `*`；估算或待核实数据使用 `!`。没有成交单、转账记录或其他来源时，不把 `!` 改成 `*`。交易说明应保留足以核对的订单号、转账标识或来源摘要；手续费金额和币种必须按原始记录填写。

每次变更后先运行 `bean-check`。不要用宽泛的全币种容差掩盖不平衡；应修正数量、价格、成本或手续费记录。
