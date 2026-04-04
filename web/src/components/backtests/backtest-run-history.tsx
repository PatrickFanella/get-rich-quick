import { Clock } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { BacktestRun } from '@/lib/api/types'

interface BacktestRunHistoryProps {
  runs: BacktestRun[]
  selectedRunId?: string
  onSelectRun: (run: BacktestRun) => void
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

export function BacktestRunHistory({ runs, selectedRunId, onSelectRun }: BacktestRunHistoryProps) {
  if (runs.length === 0) {
    return (
      <div
        className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border py-10 text-center"
        data-testid="backtest-runs-empty"
      >
        <Clock className="size-8 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">No runs yet</p>
      </div>
    )
  }

  return (
    <div className="overflow-x-auto" data-testid="backtest-runs-table">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border font-mono text-[11px] uppercase tracking-[0.14em] text-muted-foreground">
            <th className="px-3 py-2 text-left">Run Date</th>
            <th className="px-3 py-2 text-left">Duration</th>
            <th className="px-3 py-2 text-right">Total Return</th>
            <th className="px-3 py-2 text-right">Sharpe</th>
            <th className="px-3 py-2 text-right">Max Drawdown</th>
          </tr>
        </thead>
        <tbody>
          {runs.map((run) => (
            <tr
              key={run.id}
              onClick={() => onSelectRun(run)}
              className={cn(
                'cursor-pointer border-b border-border transition-colors hover:bg-accent/45',
                selectedRunId === run.id && 'bg-primary/10',
              )}
              data-testid={`backtest-run-row-${run.id}`}
            >
              <td className="px-3 py-2">{new Date(run.run_timestamp).toLocaleDateString()}</td>
              <td className="px-3 py-2 font-mono text-xs">{run.duration}</td>
              <td className="px-3 py-2 text-right font-mono text-xs">
                {formatPct(run.metrics.total_return)}
              </td>
              <td className="px-3 py-2 text-right font-mono text-xs">
                {formatRatio(run.metrics.sharpe_ratio)}
              </td>
              <td className="px-3 py-2 text-right font-mono text-xs">
                {formatPct(run.metrics.max_drawdown)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
