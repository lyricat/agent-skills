#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/terminal-table.sh"
ledger="$PWD/main.bean"
output_style='auto'

usage() {
  printf '%s\n' 'Usage: bal.sh [--ledger FILE] [--pretty|--plain]'
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

uncosted_accounts_csv="$(
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

summary_rows="$(
  {
    printf '@all\n%s\n' "$all_accounts_csv"
    printf '@costed\n%s\n' "$costed_accounts_csv"
  } | awk -F, '
    function trim(value) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      return value
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
      if (units > 0) costed_long[currency] += units
      if (units < 0) costed_short[currency] -= units
    }
    END {
      for (currency in seen) {
        held = assets[currency] + 0
        debt = -(liabilities[currency] + 0)
        if (debt == 0) debt = 0
        net = held - debt
        if (currency ~ /^(USD|USDC|USDS|USDT)$/) {
          printf "%s|stable|%.2f|%.2f|%.2f|0|1.00\n", currency, held, debt, net
        } else if ((costed_long[currency] + costed_short[currency]) > 0 && (costed[currency] + 0) == 0) {
          printf "%s|hedged|%.8f|%.8f|%.8f|0|0|%.8f|%.8f\n", currency, held, debt, net, costed_long[currency], costed_short[currency]
        } else if ((costed[currency] + 0) != 0) {
          average = cost_basis[currency] / costed[currency]
          printf "%s|costed|%.8f|%.8f|%.8f|%.8f|%.2f|%.8f|%.8f\n", currency, held, debt, net, costed[currency], average, costed_long[currency], costed_short[currency]
        } else {
          printf "%s|uncosted|%.8f|%.8f|%.8f|0|0|0|0\n", currency, held, debt, net
        }
      }
    }
  ' | sort -t '|' -k1,1
)"

if [[ "$pretty" == false ]]; then
  if ((pending_count > 0)); then
    printf '待核实交易：%d 笔\n\n' "$pending_count"
  fi

  printf '[净持仓]\n'
  if [[ -z "$summary_rows" ]]; then
    printf '无\n'
  else
    awk -F'|' '
      $2 == "stable" {
        printf "%s | 账户持有 %.2f | 负债 %.2f | 净持仓 %.2f | 均价 1.00 USDT（按面值）\n", $1, $3, $4, $5
        next
      }
      $2 == "costed" {
        printf "%s | 账户持有 %.8f | 负债 %.8f | 净持仓 %.8f | 成本持仓 %.8f | 均价 %.2f USDT\n", $1, $3, $4, $5, $6, $7
        next
      }
      $2 == "hedged" {
        printf "%s | 账户持有 %.8f | 负债 %.8f | 净持仓 %.8f | 成本多头 %.8f | 成本空头 %.8f | 均价 不适用（多空数量相抵）\n", $1, $3, $4, $5, $8, $9
        next
      }
      {
        printf "%s | 账户持有 %.8f | 负债 %.8f | 净持仓 %.8f | 成本持仓 0.00000000 | 均价 无法计算\n", $1, $3, $4, $5
      }
    ' <<< "$summary_rows"
  fi

  printf '\n[分账户：有成本记录]\n'
  if [[ "$(awk 'END { print NR }' <<< "$costed_accounts_csv")" -le 1 ]]; then
    printf '无\n'
  else
    awk -F, '
      NR == 1 { next }
      {
        printf "%s | %s | 持仓 %.8f | 均价 %.2f USDT | 成本 %.2f USDT\n", $2, $1, $3, $5, $4
      }
    ' <<< "$costed_accounts_csv"
  fi

  printf '\n[分账户：无成本记录]\n'
  if [[ "$(awk 'END { print NR }' <<< "$uncosted_accounts_csv")" -le 1 ]]; then
    printf '无\n'
  else
    awk -F, '
      NR == 1 { next }
      {
        printf "%s | %s | 持仓 %.8f | 均价 无法计算\n", $2, $1, $3
      }
    ' <<< "$uncosted_accounts_csv"
  fi

  printf '\n[稳定币分账户]\n'
  if [[ "$(awk 'END { print NR }' <<< "$stable_accounts_csv")" -le 1 ]]; then
    printf '无\n'
  else
    awk -F, '
      NR == 1 { next }
      {
        printf "%s | %s | 持仓 %.2f | 均价 1.00 USDT（按面值）\n", $2, $1, $3
      }
    ' <<< "$stable_accounts_csv"
  fi
  exit 0
fi

printf '%s · 账户余额\n' "$ledger_title"
printf '账本 %s\n' "${ledger##*/}"
if ((pending_count > 0)); then
  printf '待核实交易 %d 笔\n' "$pending_count"
fi

printf '\n净持仓\n'
if [[ -z "$summary_rows" ]]; then
  printf '无\n'
else
  {
    printf '资产|账户持有|负债|净持仓|成本持仓|均价（USDT）\n'
    awk -F'|' '
      $2 == "stable" { printf "%s|%.2f|%.2f|%.2f|—|1.00\n", $1, $3, $4, $5; next }
      $2 == "costed" { printf "%s|%.8f|%.8f|%.8f|%.8f|%.2f\n", $1, $3, $4, $5, $6, $7; next }
      $2 == "hedged" { printf "%s|%.8f|%.8f|%.8f|多 %.8f / 空 %.8f|不适用（多空相抵）\n", $1, $3, $4, $5, $8, $9; next }
      { printf "%s|%.8f|%.8f|%.8f|0.00000000|无法计算\n", $1, $3, $4, $5 }
    ' <<< "$summary_rows"
  } | draw_table '2,3,4,5,6'
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

printf '\n分账户 · 无成本记录\n'
if [[ "$(awk 'END { print NR }' <<< "$uncosted_accounts_csv")" -le 1 ]]; then
  printf '无\n'
else
  {
    printf '资产|账户|持仓|均价\n'
    awk -F, 'NR > 1 { printf "%s|%s|%.8f|无法计算\n", $2, $1, $3 }' <<< "$uncosted_accounts_csv"
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
