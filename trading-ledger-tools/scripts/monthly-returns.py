#!/usr/bin/env python3
"""Generate a monthly fund-performance chart from a Beancount ledger."""

from __future__ import annotations

import argparse
import calendar
import csv
import json
import math
import sys
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from datetime import date, datetime, time, timedelta, timezone
from decimal import Decimal
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen
from xml.sax.saxutils import escape

from beancount import loader
from beancount.core import data, prices
from beancount.parser import printer


STABLE_CURRENCIES = {"USD", "USDC", "USDS", "USDT"}
CAPITAL_IN_ACCOUNTS = (
    "Equity:OpeningBalances",
    "Equity:Contributions",
    "Equity:Adjustments",
)
DISTRIBUTION_ACCOUNT = "Equity:Distributions"
ZERO = Decimal("0")
TWO_PLACES = Decimal("0.01")


class PerformanceError(Exception):
    pass


@dataclass
class Snapshot:
    month: str
    day: date
    balances: dict[str, Decimal]
    capital_postings: list[tuple[date, str, Decimal]]
    distribution_postings: list[tuple[date, str, Decimal]]


@dataclass
class PerformanceRow:
    month: str
    nav: Decimal
    capital_in: Decimal
    distributions: Decimal
    monthly_profit: Decimal
    cumulative_profit: Decimal
    cumulative_return: Decimal | None


def account_is(account: str, parent: str) -> bool:
    return account == parent or account.startswith(parent + ":")


def month_starts(first_day: date, last_day: date):
    current = first_day.replace(day=1)
    final = last_day.replace(day=1)
    while current <= final:
        yield current
        if current.month == 12:
            current = date(current.year + 1, 1, 1)
        else:
            current = date(current.year, current.month + 1, 1)


def fetch_binance_daily_closes(
    symbol: str, first_month: date, as_of: date
) -> dict[str, Decimal]:
    start = datetime.combine(first_month, time.min, tzinfo=timezone.utc)
    end = datetime.combine(as_of + timedelta(days=1), time.min, tzinfo=timezone.utc)
    cursor_ms = int(start.timestamp() * 1000)
    end_ms = int(end.timestamp() * 1000) - 1
    monthly_closes: dict[str, Decimal] = {}

    while cursor_ms <= end_ms:
        query = urlencode(
            {
                "symbol": f"{symbol}USDT",
                "interval": "1d",
                "startTime": cursor_ms,
                "endTime": end_ms,
                "limit": 1000,
            }
        )
        request = Request(
            f"https://data-api.binance.vision/api/v3/klines?{query}",
            headers={"User-Agent": "monthly-returns/1.0"},
        )
        try:
            with urlopen(request, timeout=20) as response:
                payload = json.load(response)
        except (HTTPError, URLError, TimeoutError, json.JSONDecodeError) as error:
            raise PerformanceError(
                f"无法获取 {symbol}USDT 的 Binance 历史价格：{error}"
            ) from error

        if not isinstance(payload, list):
            raise PerformanceError(
                f"Binance 返回了无效的 {symbol}USDT 历史价格"
            )
        if not payload:
            break

        for candle in payload:
            if not isinstance(candle, list) or len(candle) < 5:
                raise PerformanceError(
                    f"Binance 返回了无效的 {symbol}USDT K 线"
                )
            candle_day = datetime.fromtimestamp(
                int(candle[0]) / 1000, tz=timezone.utc
            ).date()
            if candle_day <= as_of:
                monthly_closes[candle_day.strftime("%Y-%m")] = Decimal(str(candle[4]))

        next_cursor = int(payload[-1][0]) + 24 * 60 * 60 * 1000
        if next_cursor <= cursor_ms:
            raise PerformanceError(f"Binance 的 {symbol}USDT 历史价格未能继续翻页")
        cursor_ms = next_cursor
        if len(payload) < 1000:
            break

    return monthly_closes


def build_snapshots(entries, start_day: date, as_of: date) -> list[Snapshot]:
    transactions = [
        entry
        for entry in entries
        if isinstance(entry, data.Transaction) and entry.date <= as_of
    ]
    balances: defaultdict[str, Decimal] = defaultdict(Decimal)
    transaction_index = 0
    capital_by_month: defaultdict[
        str, list[tuple[date, str, Decimal]]
    ] = defaultdict(list)
    distributions_by_month: defaultdict[
        str, list[tuple[date, str, Decimal]]
    ] = defaultdict(list)

    snapshots = []
    for month_start in month_starts(start_day, as_of):
        calendar_end = date(
            month_start.year,
            month_start.month,
            calendar.monthrange(month_start.year, month_start.month)[1],
        )
        snapshot_day = min(calendar_end, as_of)
        month = month_start.strftime("%Y-%m")

        while (
            transaction_index < len(transactions)
            and transactions[transaction_index].date <= snapshot_day
        ):
            transaction = transactions[transaction_index]
            transaction_month = transaction.date.strftime("%Y-%m")
            for posting in transaction.postings:
                if posting.units is None:
                    continue
                number = posting.units.number
                currency = posting.units.currency
                if posting.account.startswith(("Assets:", "Liabilities:")):
                    balances[currency] += number
                elif any(
                    account_is(posting.account, parent)
                    for parent in CAPITAL_IN_ACCOUNTS
                ):
                    capital_by_month[transaction_month].append(
                        (transaction.date, currency, -number)
                    )
                elif account_is(posting.account, DISTRIBUTION_ACCOUNT):
                    distributions_by_month[transaction_month].append(
                        (transaction.date, currency, number)
                    )
            transaction_index += 1

        snapshots.append(
            Snapshot(
                month=month,
                day=snapshot_day,
                balances=dict(balances),
                capital_postings=capital_by_month[month],
                distribution_postings=distributions_by_month[month],
            )
        )

    return snapshots


def calculate_performance(
    snapshots: list[Snapshot],
    price_map,
    online_prices: dict[str, dict[str, Decimal]],
) -> list[PerformanceRow]:
    cumulative_capital = ZERO
    cumulative_distributions = ZERO
    previous_cumulative_profit = ZERO
    rows = []

    def market_price(currency: str, day: date, month: str) -> Decimal:
        if currency in STABLE_CURRENCIES:
            return Decimal("1")
        if month in online_prices.get(currency, {}):
            return online_prices[currency][month]

        price_day, number = prices.get_price(price_map, (currency, "USDT"), day)
        if number is None or price_day is None or price_day.strftime("%Y-%m") != month:
            raise PerformanceError(f"{month} 月末的 {currency} 缺少 USDT 价格")
        return number

    def postings_value(
        postings: list[tuple[date, str, Decimal]], month: str
    ) -> Decimal:
        total = ZERO
        for posting_day, currency, number in postings:
            if currency in STABLE_CURRENCIES:
                total += number
                continue
            price_day, price = prices.get_price(
                price_map, (currency, "USDT"), posting_day
            )
            if price is None or price_day is None:
                raise PerformanceError(
                    f"{posting_day} 的权益变动 {currency} 缺少 USDT 价格"
                )
            total += number * price
        return total

    for snapshot in snapshots:
        nav = sum(
            (
                units * market_price(currency, snapshot.day, snapshot.month)
                for currency, units in snapshot.balances.items()
                if units != ZERO
            ),
            ZERO,
        )
        capital_in = postings_value(snapshot.capital_postings, snapshot.month)
        distributions = postings_value(
            snapshot.distribution_postings, snapshot.month
        )
        cumulative_capital += capital_in
        cumulative_distributions += distributions
        cumulative_profit = nav + cumulative_distributions - cumulative_capital
        monthly_profit = cumulative_profit - previous_cumulative_profit
        cumulative_return = (
            cumulative_profit / cumulative_capital * 100
            if cumulative_capital != ZERO
            else None
        )
        rows.append(
            PerformanceRow(
                month=snapshot.month,
                nav=nav,
                capital_in=capital_in,
                distributions=distributions,
                monthly_profit=monthly_profit,
                cumulative_profit=cumulative_profit,
                cumulative_return=cumulative_return,
            )
        )
        previous_cumulative_profit = cumulative_profit

    return rows


def format_axis_amount(value: float) -> str:
    absolute = abs(value)
    if absolute >= 1_000_000:
        return f"{value / 1_000_000:.1f}m"
    if absolute >= 1_000:
        return f"{value / 1_000:.0f}k"
    return f"{value:.0f}"


def render_svg(
    rows: list[PerformanceRow], title: str, as_of: date, pending_count: int
) -> str:
    width = 1200
    height = 760
    left = 92
    right = 52
    plot_width = width - left - right
    line_top = 128
    line_bottom = 420
    bar_top = 510
    bar_bottom = 682
    count = len(rows)

    def x_position(index: int) -> float:
        if count == 1:
            return left + plot_width / 2
        return left + plot_width * index / (count - 1)

    def bounds(values: list[float]) -> tuple[float, float]:
        low = min(min(values), 0.0)
        high = max(max(values), 0.0)
        if low == high:
            margin = max(abs(low) * 0.1, 1.0)
            return low - margin, high + margin
        margin = (high - low) * 0.08
        return low - margin, high + margin

    cumulative_values = [float(row.cumulative_profit) for row in rows]
    monthly_values = [float(row.monthly_profit) for row in rows]
    cumulative_low, cumulative_high = bounds(cumulative_values)
    monthly_low, monthly_high = bounds(monthly_values)

    def y_position(value: float, low: float, high: float, top: int, bottom: int):
        return bottom - (value - low) * (bottom - top) / (high - low)

    line_points = " ".join(
        f"{x_position(index):.1f},{y_position(value, cumulative_low, cumulative_high, line_top, line_bottom):.1f}"
        for index, value in enumerate(cumulative_values)
    )
    line_zero = y_position(0, cumulative_low, cumulative_high, line_top, line_bottom)
    bar_zero = y_position(0, monthly_low, monthly_high, bar_top, bar_bottom)
    bar_step = plot_width / max(count, 1)
    bar_width = max(4.0, min(28.0, bar_step * 0.62))
    label_step = max(1, math.ceil(count / 10))
    last = rows[-1]
    return_label = (
        f"{last.cumulative_return:.2f}%"
        if last.cumulative_return is not None
        else "—"
    )
    pending_note = f" · 含 {pending_count} 笔待核实交易" if pending_count else ""

    svg = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}" role="img" aria-labelledby="title desc">',
        f'<title id="title">{escape(title)} · 月度收益</title>',
        f'<desc id="desc">截至 {as_of.isoformat()} 的累计收益曲线和月收益柱状图</desc>',
        "<defs>",
        '<linearGradient id="area" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#2563eb" stop-opacity="0.22"/><stop offset="1" stop-color="#2563eb" stop-opacity="0.02"/></linearGradient>',
        '<style>text{font-family:system-ui,-apple-system,"Noto Sans CJK SC",sans-serif;fill:#172033}.muted{fill:#657086}.axis{font-size:12px}.label{font-size:14px;font-weight:650}.grid{stroke:#dfe4ec;stroke-width:1}.zero{stroke:#9aa5b5;stroke-width:1.2}.positive{fill:#16a36a}.negative{fill:#dc4c4c}</style>',
        "</defs>",
        '<rect width="1200" height="760" rx="20" fill="#f8fafc"/>',
        f'<text x="{left}" y="48" font-size="25" font-weight="720">{escape(title)} · 月度收益</text>',
        f'<text x="{left}" y="76" class="muted" font-size="13">月末净资产变化已剔除投入、账面调整和分配 · 截至 {as_of.isoformat()}{escape(pending_note)}</text>',
        f'<text x="{left}" y="112" class="label">累计收益（USDT）</text>',
    ]

    for tick in range(5):
        value = cumulative_low + (cumulative_high - cumulative_low) * tick / 4
        y = y_position(value, cumulative_low, cumulative_high, line_top, line_bottom)
        svg.append(f'<line x1="{left}" y1="{y:.1f}" x2="{width-right}" y2="{y:.1f}" class="grid"/>')
        svg.append(
            f'<text x="{left-12}" y="{y+4:.1f}" text-anchor="end" class="axis muted">{escape(format_axis_amount(value))}</text>'
        )

    svg.extend(
        [
            f'<line x1="{left}" y1="{line_zero:.1f}" x2="{width-right}" y2="{line_zero:.1f}" class="zero"/>',
            f'<polygon points="{line_points} {x_position(count-1):.1f},{line_zero:.1f} {x_position(0):.1f},{line_zero:.1f}" fill="url(#area)"/>',
            f'<polyline points="{line_points}" fill="none" stroke="#2563eb" stroke-width="3" stroke-linejoin="round" stroke-linecap="round"/>',
        ]
    )
    last_x = x_position(count - 1)
    last_y = y_position(
        cumulative_values[-1], cumulative_low, cumulative_high, line_top, line_bottom
    )
    label_y = max(line_top + 20, min(line_bottom - 10, last_y - 14))
    svg.extend(
        [
            f'<circle cx="{last_x:.1f}" cy="{last_y:.1f}" r="5" fill="#2563eb" stroke="#fff" stroke-width="2"/>',
            f'<text x="{last_x-8:.1f}" y="{label_y:.1f}" text-anchor="end" class="label">{last.cumulative_profit:,.2f} · {return_label}</text>',
            f'<text x="{left}" y="492" class="label">月收益（USDT）</text>',
        ]
    )

    for tick in range(3):
        value = monthly_low + (monthly_high - monthly_low) * tick / 2
        y = y_position(value, monthly_low, monthly_high, bar_top, bar_bottom)
        svg.append(f'<line x1="{left}" y1="{y:.1f}" x2="{width-right}" y2="{y:.1f}" class="grid"/>')
        svg.append(
            f'<text x="{left-12}" y="{y+4:.1f}" text-anchor="end" class="axis muted">{escape(format_axis_amount(value))}</text>'
        )
    svg.append(
        f'<line x1="{left}" y1="{bar_zero:.1f}" x2="{width-right}" y2="{bar_zero:.1f}" class="zero"/>'
    )

    for index, row in enumerate(rows):
        value = float(row.monthly_profit)
        y = y_position(value, monthly_low, monthly_high, bar_top, bar_bottom)
        top = min(y, bar_zero)
        height_value = max(abs(bar_zero - y), 1.0)
        css_class = "positive" if value >= 0 else "negative"
        svg.append(
            f'<rect x="{x_position(index)-bar_width/2:.1f}" y="{top:.1f}" width="{bar_width:.1f}" height="{height_value:.1f}" rx="2" class="{css_class}"/>'
        )
        if index % label_step == 0 or index == count - 1:
            svg.append(
                f'<text x="{x_position(index):.1f}" y="{bar_bottom+24}" text-anchor="middle" class="axis muted">{row.month}</text>'
            )

    svg.extend(
        [
            f'<text x="{left}" y="742" class="axis muted">累计收益 = 净资产 + 累计分配 − 累计投入；百分比为累计收益 ÷ 累计投入，不是时间加权收益率。</text>',
            "</svg>",
            "",
        ]
    )
    return "\n".join(svg)


def write_csv(path: Path, rows: list[PerformanceRow]) -> None:
    with path.open("w", encoding="utf-8", newline="") as output:
        writer = csv.DictWriter(
            output,
            fieldnames=[
                "month",
                "nav_usdt",
                "capital_in_usdt",
                "distributions_usdt",
                "monthly_profit_usdt",
                "cumulative_profit_usdt",
                "cumulative_return_pct",
            ],
        )
        writer.writeheader()
        for row in rows:
            writer.writerow(
                {
                    "month": row.month,
                    "nav_usdt": f"{row.nav.quantize(TWO_PLACES):.2f}",
                    "capital_in_usdt": f"{row.capital_in.quantize(TWO_PLACES):.2f}",
                    "distributions_usdt": f"{row.distributions.quantize(TWO_PLACES):.2f}",
                    "monthly_profit_usdt": f"{row.monthly_profit.quantize(TWO_PLACES):.2f}",
                    "cumulative_profit_usdt": f"{row.cumulative_profit.quantize(TWO_PLACES):.2f}",
                    "cumulative_return_pct": (
                        f"{row.cumulative_return.quantize(TWO_PLACES):.2f}"
                        if row.cumulative_return is not None
                        else ""
                    ),
                }
            )


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="按月计算净资产和收益，并生成 SVG 曲线与 CSV 明细。"
    )
    parser.add_argument("--ledger", default="main.bean", help="Beancount 主账本")
    parser.add_argument(
        "--output", default="monthly-returns.svg", help="SVG 输出路径"
    )
    parser.add_argument(
        "--csv", dest="csv_output", help="CSV 输出路径；默认与 SVG 同名"
    )
    parser.add_argument(
        "--as-of",
        type=date.fromisoformat,
        default=date.today(),
        help="统计截止日，格式 YYYY-MM-DD；默认今天",
    )
    parser.add_argument(
        "--offline",
        action="store_true",
        help="只使用账本价格；每个持仓月份都必须有当月价格",
    )
    return parser.parse_args()


def main() -> int:
    arguments = parse_arguments()
    ledger = Path(arguments.ledger)
    svg_path = Path(arguments.output)
    csv_path = (
        Path(arguments.csv_output)
        if arguments.csv_output
        else svg_path.with_suffix(".csv")
    )

    try:
        if not ledger.is_file():
            raise PerformanceError(f"账本不存在：{ledger}")
        if svg_path == csv_path:
            raise PerformanceError("SVG 与 CSV 不能写入同一个文件")

        entries, errors, options_map = loader.load_file(str(ledger))
        if errors:
            printer.print_errors(errors, file=sys.stderr)
            return 1

        transactions = [
            entry
            for entry in entries
            if isinstance(entry, data.Transaction)
            and entry.date <= arguments.as_of
            and any(
                posting.account.startswith(("Assets:", "Liabilities:"))
                or any(
                    account_is(posting.account, parent)
                    for parent in (*CAPITAL_IN_ACCOUNTS, DISTRIBUTION_ACCOUNT)
                )
                for posting in entry.postings
            )
        ]
        if not transactions:
            raise PerformanceError("截止日之前没有可统计的交易")

        start_day = min(entry.date for entry in transactions)
        snapshots = build_snapshots(entries, start_day, arguments.as_of)
        required_months: defaultdict[str, list[date]] = defaultdict(list)
        for snapshot in snapshots:
            for currency, units in snapshot.balances.items():
                if units != ZERO and currency not in STABLE_CURRENCIES:
                    required_months[currency].append(snapshot.day)

        online_prices: dict[str, dict[str, Decimal]] = {}
        if not arguments.offline:
            first_month = start_day.replace(day=1)
            currencies = sorted(required_months)
            if currencies:
                with ThreadPoolExecutor(
                    max_workers=min(8, len(currencies))
                ) as executor:
                    requests = {
                        executor.submit(
                            fetch_binance_daily_closes,
                            currency,
                            first_month,
                            arguments.as_of,
                        ): currency
                        for currency in currencies
                    }
                    for request in as_completed(requests):
                        currency = requests[request]
                        online_prices[currency] = request.result()

        price_map = prices.build_price_map(entries)
        rows = calculate_performance(snapshots, price_map, online_prices)
        pending_count = sum(
            1
            for entry in transactions
            if isinstance(entry, data.Transaction) and entry.flag == "!"
        )
        title = options_map.get("title") or ledger.name
        svg = render_svg(rows, title, arguments.as_of, pending_count)

        svg_path.write_text(svg, encoding="utf-8")
        write_csv(csv_path, rows)
    except (PerformanceError, OSError, ValueError) as error:
        print(f"错误：{error}", file=sys.stderr)
        return 1

    print(f"已生成 SVG：{svg_path}")
    print(f"已生成 CSV：{csv_path}")
    summary = (
        f"截至 {arguments.as_of.isoformat()}：累计收益 "
        f"{rows[-1].cumulative_profit:,.2f} USDT"
    )
    if rows[-1].cumulative_return is not None:
        summary += f"，累计简单收益率 {rows[-1].cumulative_return:.2f}%"
    print(summary)
    return 0


if __name__ == "__main__":
    sys.exit(main())
