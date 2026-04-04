import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { BacktestMetrics } from '@/lib/api/types'

interface BacktestMetricsCardProps {
  metrics: BacktestMetrics
}

function toNumber(value: number | string): number | null {
  if (typeof value === 'number') return value
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

function formatPct(value: number | string): string {
  const n = toNumber(value)
  if (n == null) return String(value)
  return `${(n * 100).toFixed(2)}%`
}

function formatRatio(value: number | string): string {
  const n = toNumber(value)
  if (n == null) return String(value)
  return n.toFixed(2)
}

function valueColor(value: number | string, invert = false): string {
  const n = toNumber(value)
  if (n == null) return 'text-yellow-500'
  if (invert) return n < 0 ? 'text-green-500' : n > 0 ? 'text-red-500' : 'text-foreground'
  return n > 0 ? 'text-green-500' : n < 0 ? 'text-red-500' : 'text-foreground'
}

interface MetricItemProps {
  label: string
  value: string
  colorClass: string
}

function MetricItem({ label, value, colorClass }: MetricItemProps) {
  return (
    <div>
      <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
        {label}
      </dt>
      <dd className={cn('mt-1 text-sm font-medium', colorClass)}>{value}</dd>
    </div>
  )
}

export function BacktestMetricsCard({ metrics }: BacktestMetricsCardProps) {
  return (
    <Card data-testid="backtest-metrics-card">
      <CardHeader>
        <CardTitle>Metrics</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <MetricItem
            label="Total Return"
            value={formatPct(metrics.total_return)}
            colorClass={valueColor(metrics.total_return)}
          />
          <MetricItem
            label="Sharpe Ratio"
            value={formatRatio(metrics.sharpe_ratio)}
            colorClass={valueColor(metrics.sharpe_ratio)}
          />
          <MetricItem
            label="Max Drawdown"
            value={formatPct(metrics.max_drawdown)}
            colorClass={valueColor(metrics.max_drawdown, true)}
          />
          <MetricItem
            label="Win Rate"
            value={formatPct(metrics.win_rate)}
            colorClass={valueColor(metrics.win_rate)}
          />
          <MetricItem
            label="Profit Factor"
            value={formatRatio(metrics.profit_factor)}
            colorClass={valueColor(metrics.profit_factor)}
          />
          <MetricItem
            label="Sortino Ratio"
            value={formatRatio(metrics.sortino_ratio)}
            colorClass={valueColor(metrics.sortino_ratio)}
          />
        </dl>
      </CardContent>
    </Card>
  )
}
