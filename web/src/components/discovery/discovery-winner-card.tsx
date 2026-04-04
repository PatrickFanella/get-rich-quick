import { Link } from 'react-router-dom'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { BacktestMetrics, DeployedStrategy } from '@/lib/api/types'

interface DiscoveryWinnerCardProps {
  winner: DeployedStrategy
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

function scoreBadgeVariant(score: number) {
  if (score > 0.7) return 'success' as const
  if (score > 0.4) return 'warning' as const
  return 'destructive' as const
}

interface MetricRowProps {
  label: string
  inSample: string
  outOfSample: string
  inSampleColor: string
  outOfSampleColor: string
}

function MetricRow({ label, inSample, outOfSample, inSampleColor, outOfSampleColor }: MetricRowProps) {
  return (
    <div className="grid grid-cols-[1fr_auto_auto] items-center gap-4 py-1">
      <span className="font-mono text-[11px] uppercase tracking-[0.14em] text-muted-foreground">{label}</span>
      <span className={cn('text-right text-sm font-medium tabular-nums', inSampleColor)}>{inSample}</span>
      <span className={cn('text-right text-sm font-medium tabular-nums', outOfSampleColor)}>{outOfSample}</span>
    </div>
  )
}

function getStrategyName(config: unknown): string {
  if (config && typeof config === 'object' && 'name' in config && typeof (config as Record<string, unknown>).name === 'string') {
    return (config as Record<string, string>).name
  }
  return 'Strategy'
}

function MetricsComparison({ inSample, outOfSample }: { inSample: BacktestMetrics; outOfSample: BacktestMetrics }) {
  const rows: { label: string; invert?: boolean; format: (v: number | string) => string; inSample: number | string; outOfSample: number | string }[] = [
    { label: 'Total Return', format: formatPct, inSample: inSample.total_return, outOfSample: outOfSample.total_return },
    { label: 'Sharpe', format: formatRatio, inSample: inSample.sharpe_ratio, outOfSample: outOfSample.sharpe_ratio },
    { label: 'Sortino', format: formatRatio, inSample: inSample.sortino_ratio, outOfSample: outOfSample.sortino_ratio },
    { label: 'Max Drawdown', format: formatPct, invert: true, inSample: inSample.max_drawdown, outOfSample: outOfSample.max_drawdown },
  ]

  return (
    <div>
      <div className="grid grid-cols-[1fr_auto_auto] gap-4 border-b border-border pb-1.5">
        <span />
        <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-muted-foreground">In-Sample</span>
        <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-muted-foreground">Out-of-Sample</span>
      </div>
      {rows.map((row) => (
        <MetricRow
          key={row.label}
          label={row.label}
          inSample={row.format(row.inSample)}
          outOfSample={row.format(row.outOfSample)}
          inSampleColor={valueColor(row.inSample, row.invert)}
          outOfSampleColor={valueColor(row.outOfSample, row.invert)}
        />
      ))}
    </div>
  )
}

export function DiscoveryWinnerCard({ winner }: DiscoveryWinnerCardProps) {
  const name = getStrategyName(winner.config)

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2.5">
            <CardTitle className="text-base">{winner.ticker}</CardTitle>
            <span className="text-sm text-muted-foreground">{name}</span>
          </div>
          <Badge variant={scoreBadgeVariant(winner.score)}>
            {winner.score.toFixed(2)}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <MetricsComparison inSample={winner.in_sample} outOfSample={winner.out_of_sample} />
        <Link
          to={`/strategies/${winner.strategy_id}`}
          className="inline-flex items-center text-sm font-medium text-primary hover:underline"
        >
          View Strategy &rarr;
        </Link>
      </CardContent>
    </Card>
  )
}
