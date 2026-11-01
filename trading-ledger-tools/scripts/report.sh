#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/terminal-table.sh"
ledger="$PWD/main.bean"
declare -A supplied_prices=()
output_style='auto'

usage() {
  printf '%s\n' \
    'Usage: report.sh [--ledger FILE] [--price SYMBOL=PRICE]... [--pretty|--plain]' \
    '' \
    '默认读取当前目录的 main.bean，并从 Binance 获取未指定非稳定币的 USDT 实时价格。' \
    '示例：report.sh --ledger trading/main.bean --price BTC=72000 --price ETH=2300 --price SOL=87'
}

while (($# > 0)); do
  case "$1" in
    --ledger)
      if (($# < 2)); then
        echo '--ledger 缺少文件路径' >&2
        exit 2
      fi
      ledger="$2"
      shift 2
      ;;
    --price)
      if (($# < 2)) || [[ "$2" != *=* ]]; then
        echo '--price 必须使用 SYMBOL=PRICE 格式' >&2
        exit 2
      fi
      symbol="${2%%=*}"
      price="${2#*=}"
      symbol="${symbol^^}"
      if [[ ! "$symbol" =~ ^[A-Z0-9]+$ ]] ||
        [[ ! "$price" =~ ^[0-9]+([.][0-9]+)?$ ]] ||
        ! awk -v value="$price" 'BEGIN { exit !(value > 0) }'; then
        echo "无效价格：$2" >&2
        exit 2
      fi
      supplied_prices["$symbol"]="$price"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --pretty)
      output_style='pretty'
      shift
      ;;
    --plain)
      output_style='plain'
      shift
      ;;
    *)
      echo "未知参数：$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

pretty=false
if [[ "$output_style" == 'pretty' ]] ||
  [[ "$output_style" == 'auto' && -t 1 && -n "$(command -v column 2>/dev/null)" ]]; then
  if ! command -v column >/dev/null 2>&1; then
    echo '美化表格需要命令：column' >&2
    exit 1
  fi
  pretty=true
fi

if [[ ! -f "$ledger" ]]; then
  echo "账本不存在：$ledger" >&2
  exit 1
fi

for required_command in bean-check bean-query awk sort; do
  if ! command -v "$required_command" >/dev/null 2>&1; then
    echo "缺少命令：$required_command" >&2
    exit 1
  fi
done

bean-check "$ledger"

pending_transactions_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT id
     WHERE flag = "!"
     GROUP BY id
     ORDER BY id' \
    | tr -d '\r'
)"
pending_count="$(awk 'NR > 1 { count++ } END { print count + 0 }' <<< "$pending_transactions_csv")"
realized_total_label='历史净已实现盈亏'
if ((pending_count > 0)); then
  realized_total_label='历史净已实现盈亏（含待核实交易）'
fi

ledger_title="$(
  awk -F'"' '
    $1 ~ /^[[:space:]]*option[[:space:]]*$/ && $2 == "title" {
      print $4
      exit
    }
  ' "$ledger"
)"
if [[ -z "$ledger_title" ]]; then
  ledger_title="${ledger##*/}"
fi

all_accounts_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT account, currency, SUM(number) AS units
     WHERE account ~ "^(Assets|Liabilities):"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY currency, account' \
    | tr -d '\r'
)"

costed_accounts_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT
       account,
       currency,
       SUM(number) AS units,
       NUMBER(ONLY("USDT", SUM(COST(position)))) AS cost_basis,
       ROUND(SAFEDIV(NUMBER(ONLY("USDT", SUM(COST(position)))), SUM(number)), 2) AS average_price
     WHERE account ~ "^(Assets|Liabilities):"
       AND cost_currency = "USDT"
       AND currency !~ "^(USD|USDC|USDS|USDT)$"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY currency, account' \
    | tr -d '\r'
)"

unknown_cost_accounts_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT account, currency, SUM(number) AS units
     WHERE account ~ "^(Assets|Liabilities):"
       AND (cost_currency IS NULL OR cost_currency != "USDT")
       AND currency !~ "^(USD|USDC|USDS|USDT)$"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY currency, account' \
    | tr -d '\r'
)"

stable_accounts_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT account, currency, SUM(number) AS units
     WHERE account ~ "^(Assets|Liabilities):"
       AND currency ~ "^(USD|USDC|USDS|USDT)$"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY currency, account' \
    | tr -d '\r'
)"

exchange_accounts_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT account, currency, SUM(number) AS units
     WHERE account = "Assets:BN:Cash" OR account ~ "^Assets:BN:Spot(:.*)?$"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY currency, account' \
    | tr -d '\r'
)"

realized_yearly_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT
       YEAR(date) AS year,
       ROUND(-NUMBER(ONLY("USDT", SUM(CONVERT(position, "USDT")))), 2) AS net_usdt
     WHERE account = "Income:Gains"
       OR account ~ "^Income:Gains:"
       OR account ~ "^Expenses:.*Fees$"
     GROUP BY YEAR(date)
     ORDER BY YEAR(date)' \
    | tr -d '\r'
)"

gross_realized="$(
  bean-query --format csv "$ledger" \
    'SELECT
       ROUND(-NUMBER(ONLY("USDT", SUM(CONVERT(position, "USDT")))), 2) AS gross_usdt
     WHERE account = "Income:Gains" OR account ~ "^Income:Gains:"' \
    | awk -F, 'NR == 2 { value = $1 } END { printf "%.2f", value + 0 }'
)"

converted_fees_usdt="$(
  bean-query --format csv "$ledger" \
    'SELECT
       ROUND(NUMBER(ONLY("USDT", SUM(CONVERT(position, "USDT")))), 2) AS fees_usdt
     WHERE account ~ "^Expenses:.*Fees$"' \
    | awk -F, 'NR == 2 { value = $1 } END { printf "%.2f", value + 0 }'
)"

non_usdt_fees_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT
       currency,
       SUM(number) AS units,
       NUMBER(ONLY("USDT", SUM(CONVERT(position, "USDT")))) AS converted_usdt
     WHERE currency != "USDT"
       AND account ~ "^Expenses:.*Fees$"
     GROUP BY currency
     HAVING SUM(number) != 0
     ORDER BY currency' \
    | tr -d '\r'
)"
unconverted_fee_count="$(
  awk -F, 'NR > 1 && ($2 + 0) != 0 && ($3 + 0) == 0 { count++ } END { print count + 0 }' \
    <<< "$non_usdt_fees_csv"
)"
if ((unconverted_fee_count > 0)); then
  if ((pending_count > 0)); then
    realized_total_label='历史净已实现盈亏（含待核实交易，另有未折算手续费）'
  else
    realized_total_label='历史净已实现盈亏（另有未折算手续费）'
  fi
fi

realized_total_csv="$(bean-query --format csv "$ledger" < "$script_dir/realized.bql" | tr -d '\r')"
realized_total_row="$(awk -F, 'NR == 2 { print }' <<< "$realized_total_csv")"
IFS=, read -r _ realized_start realized_end net_realized <<< "$realized_total_row"
realized_start="${realized_start:-—}"
realized_end="${realized_end:-—}"
net_realized="${net_realized:-0}"

contributions_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT account, currency, SUM(number) AS units
     WHERE account = "Equity:Contributions"
       OR account ~ "^Equity:Contributions:"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY account, currency' \
    | tr -d '\r'
)"

opening_balances_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT account, currency, SUM(number) AS units
     WHERE account = "Equity:OpeningBalances"
       OR account ~ "^Equity:OpeningBalances:"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY account, currency' \
    | tr -d '\r'
)"

adjustments_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT account, currency, SUM(number) AS units
     WHERE account = "Equity:Adjustments"
       OR account ~ "^Equity:Adjustments:"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY account, currency' \
    | tr -d '\r'
)"

distributions_csv="$(
  bean-query --format csv "$ledger" \
    'SELECT account, currency, SUM(number) AS units
     WHERE account = "Equity:Distributions"
       OR account ~ "^Equity:Distributions:"
     GROUP BY account, currency
     HAVING SUM(number) != 0
     ORDER BY account, currency' \
    | tr -d '\r'
)"

dad_share="$(
  bean-query --format csv "$ledger" \
    'SELECT SUM(number) AS units
     WHERE account = "Equity:Ownership:BN:Dad"
       AND currency = "ALBNSHARE"' \
    | awk -F, '
        NR == 2 { gsub(/[[:space:]]/, "", $1); value = $1 }
        END { print value == "" ? "0" : value }
      '
)"

dad_capital="$(
  bean-query --format csv "$ledger" \
    'SELECT SUM(number) AS units
     WHERE account = "Equity:Basis:BN:Dad"
       AND currency ~ "^(USD|USDC|USDS|USDT)$"' \
    | awk -F, 'NR == 2 { value = -$1 } END { printf "%.10f", value + 0 }'
)"

dad_dividends="$(
  bean-query --format csv "$ledger" \
    'SELECT SUM(number) AS units
     WHERE account = "Equity:RealizedDividends:BN:Dad"
       AND currency ~ "^(USD|USDC|USDS|USDT)$"' \
    | awk -F, 'NR == 2 { value = $1 } END { printf "%.10f", value + 0 }'
)"

mom_share="$(
  bean-query --format csv "$ledger" \
    'SELECT SUM(number) AS units
     WHERE account = "Equity:Ownership:BN:Mom"
       AND currency = "ALBNSHARE"' \
    | awk -F, '
        NR == 2 { gsub(/[[:space:]]/, "", $1); value = $1 }
        END { print value == "" ? "0" : value }
      '
)"

mom_capital="$(
  bean-query --format csv "$ledger" \
    'SELECT SUM(number) AS units
     WHERE account = "Equity:Basis:BN:Mom"
       AND currency ~ "^(USD|USDC|USDS|USDT)$"' \
    | awk -F, 'NR == 2 { value = -$1 } END { printf "%.10f", value + 0 }'
)"

mom_equity_enabled=false
if awk -v share="$mom_share" -v capital="$mom_capital" \
  'BEGIN { exit !((share + 0) != 0 || (capital + 0) != 0) }'; then
  mom_equity_enabled=true
fi

mom_share_active=false
if awk -v share="$mom_share" 'BEGIN { exit !((share + 0) > 0) }'; then
  mom_share_active=true
fi

if [[ "$mom_equity_enabled" == true ]] &&
  ! awk -v share="$mom_share" 'BEGIN { exit !((share + 0) >= 0 && (share + 0) <= 1) }'; then
  echo "母亲交易所权益比例无效：$mom_share" >&2
  exit 1
fi

dad_equity_enabled=false
if awk -v share="$dad_share" -v capital="$dad_capital" -v dividends="$dad_dividends" \
  'BEGIN { exit !((share + 0) != 0 || (capital + 0) != 0 || (dividends + 0) != 0) }'; then
  dad_equity_enabled=true
fi

dad_share_active=false
if awk -v share="$dad_share" 'BEGIN { exit !((share + 0) > 0) }'; then
  dad_share_active=true
fi

if [[ "$dad_equity_enabled" == true ]] &&
  ! awk -v share="$dad_share" 'BEGIN { exit !((share + 0) >= 0 && (share + 0) <= 1) }'; then
  echo "父亲交易所权益比例无效：$dad_share" >&2
  exit 1
fi

symbols=()
while IFS= read -r currency; do
  [[ -n "$currency" ]] && symbols+=("$currency")
done < <(
  awk -F, '
    NR == 1 { next }
    {
      currency = $2
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", currency)
      if (currency !~ /^(USD|USDC|USDS|USDT)$/ && !seen[currency]++) print currency
    }
  ' <<< "$all_accounts_csv" | sort
)

declare -A prices=()
used_supplied_prices=false
used_binance_prices=false
live_dependencies_checked=false

for currency in "${symbols[@]}"; do
  if [[ -n "${supplied_prices[$currency]+set}" ]]; then
    prices["$currency"]="${supplied_prices[$currency]}"
    used_supplied_prices=true
    continue
  fi

  if [[ "$live_dependencies_checked" == false ]]; then
    for required_command in curl jq; do
      if ! command -v "$required_command" >/dev/null 2>&1; then
        echo "获取实时价格需要命令：$required_command" >&2
        exit 1
      fi
    done
    live_dependencies_checked=true
  fi

  ticker="${currency}USDT"
  if ! ticker_json="$(curl -fsSL "https://data-api.binance.vision/api/v3/ticker/price?symbol=$ticker")"; then
    echo "无法获取 $ticker 的 Binance 价格；可用 --price $currency=PRICE 指定" >&2
    exit 1
  fi
  if ! current_price="$(jq -er --arg ticker "$ticker" 'select(.symbol == $ticker) | .price | tonumber' <<< "$ticker_json")"; then
    echo "Binance 返回了无效的 $ticker 价格" >&2
    exit 1
  fi
  prices["$currency"]="$current_price"
  used_binance_prices=true
done

if [[ "$used_supplied_prices" == true && "$used_binance_prices" == true ]]; then
  price_source='命令行 + Binance'
elif [[ "$used_supplied_prices" == true ]]; then
  price_source='命令行'
elif [[ "$used_binance_prices" == true ]]; then
  price_source='Binance'
else
  price_source='仅稳定币'
fi

price_spec=''
for currency in "${symbols[@]}"; do
  price_spec+="$currency=${prices[$currency]};"
done

market_time=''
if [[ "$used_binance_prices" == true ]]; then
  market_time="$(date '+%Y-%m-%d %H:%M:%S %Z')"
fi

position_rows="$(
  {
    printf '@all\n%s\n' "$all_accounts_csv"
    printf '@costed\n%s\n' "$costed_accounts_csv"
  } | awk -F, -v price_spec="$price_spec" '
    function trim(value) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      return value
    }
    function abs(value) { return value < 0 ? -value : value }
    BEGIN {
      count = split(price_spec, pairs, ";")
      for (i = 1; i <= count; i++) {
        if (pairs[i] == "") continue
        split(pairs[i], pair, "=")
        prices[pair[1]] = pair[2] + 0
      }
    }
    /^@all$/ { section = "all"; next }
    /^@costed$/ { section = "costed"; next }
    $1 == "account" { next }
    section == "all" && NF >= 3 {
      account = trim($1)
      currency = trim($2)
      units = $3 + 0
      seen[currency] = 1
      if (account ~ /^Assets:/) assets[currency] += units
      if (account ~ /^Liabilities:/) liabilities[currency] += units
      next
    }
    section == "costed" && NF >= 4 {
      currency = trim($2)
      units = $3 + 0
      costed[currency] += units
      cost_basis[currency] += $4
      gross_cost_basis[currency] += abs($4 + 0)
      if (units > 0) costed_long[currency] += units
      if (units < 0) costed_short[currency] -= units
    }
    END {
      for (currency in seen) {
        held = assets[currency] + 0
        debt = -(liabilities[currency] + 0)
        if (debt == 0) debt = 0
        net = held - debt
        stable = currency ~ /^(USD|USDC|USDS|USDT)$/
        price = stable ? 1 : prices[currency]
        market_value = net * price
        costed_units = costed[currency] + 0
        basis = cost_basis[currency] + 0
        average = costed_units == 0 ? 0 : basis / costed_units
        unrealized = costed_units * price - basis
        unknown_units = net - costed_units
        if (stable) {
          kind = "stable"
        } else if ((costed_long[currency] + costed_short[currency]) > 0 && costed_units == 0) {
          kind = "hedged"
        } else if (costed_units != 0) {
          kind = "costed"
        } else {
          kind = "uncosted"
        }
        printf "%s|%s|%.10f|%.10f|%.10f|%.10f|%.10f|%.10f|%.10f|%.10f|%.10f|%.10f|%.10f|%.10f|%.10f\n", currency, kind, held, debt, net, costed_units, basis, average, price, market_value, unrealized, unknown_units, costed_long[currency], costed_short[currency], gross_cost_basis[currency]
      }
    }
  ' | sort -t '|' -k1,1
)"

totals_row="$(
  awk -F'|' '
    function abs(value) { return value < 0 ? -value : value }
    {
      total += $10
      if ($2 == "stable") {
        stable += $10
      } else {
        known += $6 * $9
        unknown += $12 * $9
        unrealized += $11
        risk_cost += $15
      }
    }
    END {
      rate = risk_cost == 0 ? 0 : 100 * unrealized / risk_cost
      printf "%.10f|%.10f|%.10f|%.10f|%.10f|%.10f", total, known, unknown, stable, unrealized, rate
    }
  ' <<< "$position_rows"
)"
IFS='|' read -r estimated_total known_market_value unknown_market_value stable_market_value known_unrealized known_return_rate <<< "$totals_row"

equity_input="$(
  printf '@opening\n%s\n' "$opening_balances_csv"
  printf '@contributions\n%s\n' "$contributions_csv"
  printf '@adjustments\n%s\n' "$adjustments_csv"
  printf '@distributions\n%s\n' "$distributions_csv"
)"

equity_totals="$(
  awk -F, '
    function trim(value) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      return value
    }
    /^@/ { section = substr($0, 2); next }
    $1 == "account" { next }
    NF >= 3 {
      currency = trim($2)
      if (currency !~ /^(USD|USDC|USDS|USDT)$/) next
      units = $3 + 0
      amount = section == "distributions" ? units : -units
      totals[section] += amount
    }
    END {
      printf "%.10f|%.10f|%.10f|%.10f", \
        totals["opening"], totals["contributions"], \
        totals["adjustments"], totals["distributions"]
    }
  ' <<< "$equity_input"
)"
IFS='|' read -r opening_stable contributed_stable adjusted_stable distributed_stable <<< "$equity_totals"

unconverted_equity_rows="$(
  awk -F, '
    function trim(value) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      return value
    }
    /^@/ { section = substr($0, 2); next }
    $1 == "account" { next }
    NF >= 3 {
      currency = trim($2)
      if (currency ~ /^(USD|USDC|USDS|USDT)$/) next
      units = $3 + 0
      amount = section == "distributions" ? units : -units
      values[section, currency] += amount
    }
    END {
      for (key in values) {
        split(key, parts, SUBSEP)
        if (values[key] != 0) {
          printf "%s|%s|%.8f\n", parts[1], parts[2], values[key]
        }
      }
    }
  ' <<< "$equity_input" | sort -t '|' -k1,1 -k2,2
)"

current_plus_distributions="$(
  awk -v current="$estimated_total" -v distributed="$distributed_stable" \
    'BEGIN { printf "%.2f", current + distributed }'
)"

if [[ "$dad_equity_enabled" == true || "$mom_equity_enabled" == true ]]; then
  exchange_total="$(
    awk -F, -v price_spec="$price_spec" '
      function trim(value) {
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
        return value
      }
      BEGIN {
        count = split(price_spec, pairs, ";")
        for (i = 1; i <= count; i++) {
          if (pairs[i] == "") continue
          split(pairs[i], pair, "=")
          prices[pair[1]] = pair[2] + 0
        }
      }
      NR == 1 { next }
      NF >= 3 {
        currency = trim($2)
        units = $3 + 0
        stable = currency ~ /^(USD|USDC|USDS|USDT)$/
        price = stable ? 1 : prices[currency]
        total += units * price
      }
      END { printf "%.10f", total + 0 }
    ' <<< "$exchange_accounts_csv"
  )"

  dad_metrics="$(
    awk \
      -v exchange="$exchange_total" \
      -v share="$dad_share" \
      -v capital="$dad_capital" \
      -v dividends="$dad_dividends" \
      'BEGIN {
        if (share > 0) {
          current = exchange * share
          undistributed = current - capital
        } else {
          current = capital
          undistributed = 0
        }
        total_return = undistributed + dividends
        printf "%.10f|%.12f|%.10f|%.10f", current, share * 100, undistributed, total_return
      }'
  )"
  IFS='|' read -r dad_current_equity dad_share_pct dad_undistributed dad_total_return \
    <<< "$dad_metrics"

  if [[ "$mom_equity_enabled" == true ]]; then
    mom_metrics="$(
      awk \
        -v exchange="$exchange_total" \
        -v share="$mom_share" \
        -v capital="$mom_capital" \
        'BEGIN {
          if (share > 0) {
            current = exchange * share
            undistributed = current - capital
          } else {
            current = capital
            undistributed = 0
          }
          total_return = undistributed
          printf "%.10f|%.12f|%.10f|%.10f", current, share * 100, undistributed, total_return
        }'
    )"
    IFS='|' read -r mom_current_equity mom_share_pct mom_undistributed mom_total_return \
      <<< "$mom_metrics"
  fi
fi

if [[ "$pretty" == false ]]; then
  printf '账本：%s\n' "$ledger_title"
  printf '账本校验：通过\n'
  if ((pending_count > 0)); then
    printf '待核实交易：%d 笔\n' "$pending_count"
  fi
  printf '价格来源：%s\n' "$price_source"
  if [[ -n "$market_time" ]]; then
    printf '行情时间：%s\n' "$market_time"
  fi

  printf '\n[净持仓与估值]\n'
  if [[ -z "$position_rows" ]]; then
    printf '无\n'
  else
    awk -F'|' '
      $2 == "stable" {
        printf "%s | 账户持有 %.2f | 负债 %.2f | 净持仓 %.2f | 均价 1.00 USDT（按面值） | 净值 %.2f USDT\n", $1, $3, $4, $5, $10
        next
      }
      $2 == "costed" {
        printf "%s | 账户持有 %.8f | 负债 %.8f | 净持仓 %.8f | 成本持仓 %.8f | 均价 %.2f USDT | 现价 %.2f USDT | 净值 %.2f USDT | 成本持仓浮动盈亏 %+.2f USDT\n", $1, $3, $4, $5, $6, $8, $9, $10, $11
        next
      }
      $2 == "hedged" {
        printf "%s | 账户持有 %.8f | 负债 %.8f | 净持仓 %.8f | 成本多头 %.8f | 成本空头 %.8f | 均价 不适用（多空数量相抵） | 现价 %.2f USDT | 净值 %.2f USDT | 成本持仓浮动盈亏 %+.2f USDT\n", $1, $3, $4, $5, $13, $14, $9, $10, $11
        next
      }
      {
        printf "%s | 账户持有 %.8f | 负债 %.8f | 净持仓 %.8f | 成本持仓 0.00000000 | 均价 无法计算 | 现价 %.2f USDT | 净值 %.2f USDT\n", $1, $3, $4, $5, $9, $10
      }
    ' <<< "$position_rows"
  fi

  printf '\n[分账户：有成本记录]\n'
  if [[ "$(awk 'END { print NR }' <<< "$costed_accounts_csv")" -le 1 ]]; then
    printf '无\n'
  else
    awk -F, -v price_spec="$price_spec" '
      BEGIN {
        count = split(price_spec, pairs, ";")
        for (i = 1; i <= count; i++) {
          if (pairs[i] == "") continue
          split(pairs[i], pair, "=")
          prices[pair[1]] = pair[2] + 0
        }
      }
      NR > 1 {
        currency = $2
        gsub(/[[:space:]]/, "", currency)
        value = ($3 + 0) * prices[currency]
        profit = value - ($4 + 0)
        printf "%s | %s | 持仓 %.8f | 均价 %.2f USDT | 成本 %.2f USDT | 市值 %.2f USDT | 浮动盈亏 %+.2f USDT\n", currency, $1, $3, $5, $4, value, profit
      }
    ' <<< "$costed_accounts_csv"
  fi

  printf '\n[分账户：成本无法计算]\n'
  if [[ "$(awk 'END { print NR }' <<< "$unknown_cost_accounts_csv")" -le 1 ]]; then
    printf '无\n'
  else
    awk -F, -v price_spec="$price_spec" '
      BEGIN {
        count = split(price_spec, pairs, ";")
        for (i = 1; i <= count; i++) {
          if (pairs[i] == "") continue
          split(pairs[i], pair, "=")
          prices[pair[1]] = pair[2] + 0
        }
      }
      NR > 1 {
        currency = $2
        gsub(/[[:space:]]/, "", currency)
        value = ($3 + 0) * prices[currency]
        printf "%s | %s | 持仓 %.8f | 均价 无法计算 | 市值 %.2f USDT\n", currency, $1, $3, value
      }
    ' <<< "$unknown_cost_accounts_csv"
  fi

  printf '\n[稳定币分账户]\n'
  if [[ "$(awk 'END { print NR }' <<< "$stable_accounts_csv")" -le 1 ]]; then
    printf '无\n'
  else
    awk -F, 'NR > 1 { printf "%s | %s | 持仓 %.2f | 均价 1.00 USDT（按面值）\n", $2, $1, $3 }' <<< "$stable_accounts_csv"
  fi

  printf '\n[历史已实现盈亏]\n'
  printf '统计区间 | %s 至 %s\n' "$realized_start" "$realized_end"
  awk -F, 'NR > 1 { printf "%s | 扣除已折算手续费后 %+.2f USDT\n", $1, $2 }' <<< "$realized_yearly_csv"
  printf '手续费前已实现盈亏 | %+.2f USDT\n' "$gross_realized"
  printf '已折算手续费 | %.2f USDT\n' "$converted_fees_usdt"
  printf '%s | %+.2f USDT\n' "$realized_total_label" "$net_realized"
  if [[ "$(awk 'END { print NR }' <<< "$non_usdt_fees_csv")" -gt 1 ]]; then
    awk -F, '
      NR > 1 && ($3 + 0) != 0 {
        printf "非 USDT 手续费 | %s %.8f | 按账本价格折算 %.2f USDT\n", $1, $2, $3
        next
      }
      NR > 1 {
        printf "未折算手续费 | %s %.8f\n", $1, $2
      }
    ' <<< "$non_usdt_fees_csv"
  fi
  if ((pending_count > 0)); then
    printf '说明 | 上述结果按现有记录计算，待核实交易确认后可能变化\n'
  fi

  printf '\n[权益与净资产]\n'
  printf '期初权益（稳定币按面值） | %.2f USDT\n' "$opening_stable"
  printf '累计注资（稳定币按面值） | %.2f USDT\n' "$contributed_stable"
  printf '账面调整（稳定币按面值） | %.2f USDT\n' "$adjusted_stable"
  printf '累计提取与利润分配 | %.2f USDT\n' "$distributed_stable"
  if [[ -n "$unconverted_equity_rows" ]]; then
    awk -F'|' '
      $1 == "opening" { label = "期初权益未折算" }
      $1 == "contributions" { label = "注资未折算" }
      $1 == "adjustments" { label = "账面调整未折算" }
      $1 == "distributions" { label = "提取与利润分配未折算" }
      { printf "%s | %s %.8f\n", label, $2, $3 }
    ' <<< "$unconverted_equity_rows"
  fi
  printf '当前净资产 | %.2f USDT\n' "$estimated_total"
  printf '当前净资产加累计提取 | %.2f USDT\n' "$current_plus_distributions"

  if [[ "$dad_equity_enabled" == true ]]; then
    printf '\n[父亲交易所权益]\n'
    printf '权益范围 | BN 现货（Cash + Spot:*，不含期货）\n'
    printf '交易所净资产 | %.2f USDT\n' "$exchange_total"
    if [[ "$dad_share_active" == true ]]; then
      printf '父亲当前权益比例 | %.12f%%\n' "$dad_share_pct"
    else
      printf '父亲当前权益比例 | 尚未生效\n'
    fi
    printf '父亲当前权益 | %.2f USDT\n' "$dad_current_equity"
    printf '父亲权益基数 | %.2f USDT\n' "$dad_capital"
    printf '父亲未实现/未分配收益 | %+.2f USDT\n' "$dad_undistributed"
    printf '父亲累计已实现分红 | %.2f USDT\n' "$dad_dividends"
    printf '父亲累计收益 | %+.2f USDT\n' "$dad_total_return"
  fi

  if [[ "$mom_equity_enabled" == true ]]; then
    printf '\n[母亲交易所权益]\n'
    printf '权益范围 | BN 现货（Cash + Spot:*，不含期货）\n'
    printf '交易所净资产 | %.2f USDT\n' "$exchange_total"
    if [[ "$mom_share_active" == true ]]; then
      printf '母亲当前权益比例 | %.12f%%\n' "$mom_share_pct"
    else
      printf '母亲当前权益比例 | 尚未生效\n'
    fi
    printf '母亲当前权益 | %.2f USDT\n' "$mom_current_equity"
    printf '母亲权益基数 | %.2f USDT\n' "$mom_capital"
    printf '母亲未实现/未分配收益 | %+.2f USDT\n' "$mom_undistributed"
    printf '母亲累计收益 | %+.2f USDT\n' "$mom_total_return"
  fi

  printf '\n[估值构成]\n'
  printf '有 USDT 成本记录仓位市值 | %.2f USDT\n' "$known_market_value"
  printf '成本无法计算仓位净值 | %.2f USDT\n' "$unknown_market_value"
  printf '稳定币净值 | %.2f USDT\n' "$stable_market_value"
  printf '当前净资产 | %.2f USDT\n' "$estimated_total"
  printf '有成本记录仓位浮动盈亏 | %+.2f USDT (%+.2f%%)\n' "$known_unrealized" "$known_return_rate"
  exit 0
fi

printf '%s · 资产负债报告\n' "$ledger_title"
printf '账本 %s · 行情 %s' "${ledger##*/}" "$price_source"
if [[ -n "$market_time" ]]; then
  printf ' · %s' "$market_time"
fi
if ((pending_count > 0)); then
  printf ' · 待核实 %d 笔' "$pending_count"
fi
printf '\n'

printf '\n净资产概览\n'
{
  printf '指标|金额（USDT）\n'
  printf '当前净资产|%.2f\n' "$estimated_total"
  printf '有 USDT 成本记录仓位|%.2f\n' "$known_market_value"
  printf '成本无法计算仓位|%.2f\n' "$unknown_market_value"
  printf '稳定币净值|%.2f\n' "$stable_market_value"
  printf '成本持仓浮动盈亏|%+.2f（%+.2f%%）\n' "$known_unrealized" "$known_return_rate"
  if ((pending_count > 0)); then
    printf '历史净已实现盈亏（含待核实）|%+.2f\n' "$net_realized"
  else
    printf '历史净已实现盈亏|%+.2f\n' "$net_realized"
  fi
} | draw_table '2'

printf '\n净持仓与估值\n'
if [[ -z "$position_rows" ]]; then
  printf '无\n'
else
  {
    printf '资产|账户持有|负债|净持仓|成本持仓|均价|现价|净值|成本仓浮盈亏\n'
    awk -F'|' '
      $2 == "stable" { printf "%s|%.2f|%.2f|%.2f|—|1.00|1.00|%.2f|—\n", $1, $3, $4, $5, $10; next }
      $2 == "costed" { printf "%s|%.8f|%.8f|%.8f|%.8f|%.2f|%.2f|%.2f|%+.2f\n", $1, $3, $4, $5, $6, $8, $9, $10, $11; next }
      $2 == "hedged" { printf "%s|%.8f|%.8f|%.8f|多 %.8f / 空 %.8f|不适用（多空相抵）|%.2f|%.2f|%+.2f\n", $1, $3, $4, $5, $13, $14, $9, $10, $11; next }
      { printf "%s|%.8f|%.8f|%.8f|0.00000000|无法计算|%.2f|%.2f|—\n", $1, $3, $4, $5, $9, $10 }
    ' <<< "$position_rows"
  } | draw_table '2,3,4,5,6,7,8,9'
fi

printf '\n历史已实现盈亏\n'
printf '统计区间 %s 至 %s\n' "$realized_start" "$realized_end"
{
  printf '期间|扣除已折算手续费后净额（USDT）\n'
  awk -F, 'NR > 1 { printf "%s|%+.2f\n", $1, $2 }' <<< "$realized_yearly_csv"
  printf '手续费前合计|%+.2f\n' "$gross_realized"
  printf '已折算手续费|-%0.2f\n' "$converted_fees_usdt"
  printf '历史净合计|%+.2f\n' "$net_realized"
} | draw_table '2'
if [[ "$(awk 'END { print NR }' <<< "$non_usdt_fees_csv")" -gt 1 ]]; then
  awk -F, '
    NR > 1 && ($3 + 0) != 0 {
      printf "非 USDT 手续费：%s %.8f，按账本价格折算 %.2f USDT\n", $1, $2, $3
      next
    }
    NR > 1 {
      printf "未折算手续费：%s %.8f\n", $1, $2
    }
  ' <<< "$non_usdt_fees_csv"
fi
if ((pending_count > 0)); then
  printf '上述结果按现有记录计算；%d 笔待核实交易确认后可能变化。\n' "$pending_count"
fi

printf '\n权益变动\n'
{
  printf '指标|金额（USDT）\n'
  printf '期初权益（稳定币按面值）|%.2f\n' "$opening_stable"
  printf '累计注资（稳定币按面值）|%.2f\n' "$contributed_stable"
  printf '账面调整（稳定币按面值）|%.2f\n' "$adjusted_stable"
  printf '累计提取与利润分配|%.2f\n' "$distributed_stable"
  printf '当前净资产|%.2f\n' "$estimated_total"
  printf '当前净资产加累计提取|%.2f\n' "$current_plus_distributions"
} | draw_table '2'
if [[ -n "$unconverted_equity_rows" ]]; then
  printf '\n未折算权益项目\n'
  {
    printf '类别|资产|数量\n'
    awk -F'|' '
      $1 == "opening" { label = "期初权益" }
      $1 == "contributions" { label = "注资" }
      $1 == "adjustments" { label = "账面调整" }
      $1 == "distributions" { label = "提取与利润分配" }
      { printf "%s|%s|%.8f\n", label, $2, $3 }
    ' <<< "$unconverted_equity_rows"
  } | draw_table '3'
fi

if [[ "$dad_equity_enabled" == true ]]; then
  printf '\n父亲交易所权益\n'
  {
    printf '指标|数值\n'
    printf '权益范围|BN 现货（Cash + Spot:*，不含期货）\n'
    printf '交易所净资产|%.2f USDT\n' "$exchange_total"
    if [[ "$dad_share_active" == true ]]; then
      printf '父亲当前权益比例|%.12f%%\n' "$dad_share_pct"
    else
      printf '父亲当前权益比例|尚未生效\n'
    fi
    printf '父亲当前权益|%.2f USDT\n' "$dad_current_equity"
    printf '父亲权益基数|%.2f USDT\n' "$dad_capital"
    printf '父亲未实现/未分配收益|%+.2f USDT\n' "$dad_undistributed"
    printf '父亲累计已实现分红|%.2f USDT\n' "$dad_dividends"
    printf '父亲累计收益|%+.2f USDT\n' "$dad_total_return"
  } | draw_table '2'
fi

if [[ "$mom_equity_enabled" == true ]]; then
  printf '\n母亲交易所权益\n'
  {
    printf '指标|数值\n'
    printf '权益范围|BN 现货（Cash + Spot:*，不含期货）\n'
    printf '交易所净资产|%.2f USDT\n' "$exchange_total"
    if [[ "$mom_share_active" == true ]]; then
      printf '母亲当前权益比例|%.12f%%\n' "$mom_share_pct"
    else
      printf '母亲当前权益比例|尚未生效\n'
    fi
    printf '母亲当前权益|%.2f USDT\n' "$mom_current_equity"
    printf '母亲权益基数|%.2f USDT\n' "$mom_capital"
    printf '母亲未实现/未分配收益|%+.2f USDT\n' "$mom_undistributed"
    printf '母亲累计收益|%+.2f USDT\n' "$mom_total_return"
  } | draw_table '2'
fi

printf '\n分账户 · 有成本记录\n'
if [[ "$(awk 'END { print NR }' <<< "$costed_accounts_csv")" -le 1 ]]; then
  printf '无\n'
else
  {
    printf '资产|账户|持仓|均价（USDT）|成本（USDT）\n'
    awk -F, 'NR > 1 { printf "%s|%s|%.8f|%.2f|%.2f\n", $2, $1, $3, $5, $4 }' <<< "$costed_accounts_csv"
  } | draw_table '3,4,5'
fi

printf '\n分账户 · 成本无法计算\n'
if [[ "$(awk 'END { print NR }' <<< "$unknown_cost_accounts_csv")" -le 1 ]]; then
  printf '无\n'
else
  {
    printf '资产|账户|持仓|均价\n'
    awk -F, 'NR > 1 { printf "%s|%s|%.8f|无法计算\n", $2, $1, $3 }' <<< "$unknown_cost_accounts_csv"
  } | draw_table '3'
fi

printf '\n稳定币分账户\n'
if [[ "$(awk 'END { print NR }' <<< "$stable_accounts_csv")" -le 1 ]]; then
  printf '无\n'
else
  {
    printf '资产|账户|持仓|均价（USDT）\n'
    awk -F, 'NR > 1 { printf "%s|%s|%.2f|1.00\n", $2, $1, $3 }' <<< "$stable_accounts_csv"
  } | draw_table '3,4'
fi
