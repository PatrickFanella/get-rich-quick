import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Play } from 'lucide-react'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { BacktestEquityChart } from '@/components/backtests/backtest-equity-chart'
import { BacktestMetricsCard } from '@/components/backtests/backtest-metrics-card'
import { BacktestRunHistory } from '@/components/backtests/backtest-run-history'
import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { apiClient } from '@/lib/api/client'
import type { BacktestRun } from '@/lib/api/types'

export function BacktestDetailPage() {
  const { id } = useParams<{ id: string }>()
  const queryClient = useQueryClient()
  const [selectedRun, setSelectedRun] = useState<BacktestRun | null>(null)

  const { data: config, isLoading, isError } = useQuery({
    queryKey: ['backtest-config', id],
    queryFn: () => apiClient.getBacktestConfig(id!),
    enabled: !!id,
  })

  const { data: runsData } = useQuery({
    queryKey: ['backtest-runs', { config_id: id }],
    queryFn: () => apiClient.listBacktestRuns({ backtest_config_id: id }),
    enabled: !!id,
    refetchInterval: 15_000,
  })
  const runs = runsData?.data ?? []

  const runMutation = useMutation({
    mutationFn: () => apiClient.runBacktestConfig(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['backtest-runs', { config_id: id }] })
    },
  })

  if (isLoading) {
    return (
      <div className="space-y-6" data-testid="backtest-detail-loading">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-64 animate-pulse rounded-lg border bg-muted" />
      </div>
    )
  }

  if (isError || !config) {
    return (
      <div className="space-y-4" data-testid="backtest-detail-error">
        <Link to="/backtests" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" />
          Back to backtests
        </Link>
        <Card>
          <CardContent className="py-8">
            <p className="text-center text-sm text-muted-foreground">
              Unable to load backtest config. It may have been deleted or the API server is unavailable.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-4" data-testid="backtest-detail-page">
      <PageHeader
        title={config.name}
        description={config.description || 'Backtest configuration details and run history.'}
        actions={(
          <>
            <Link
              to="/backtests"
              className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-3 py-2 text-sm text-muted-foreground transition-colors hover:border-primary/25 hover:text-foreground"
            >
              <ArrowLeft className="size-4" />
              Back
            </Link>
            <Button
              variant="outline"
              onClick={() => runMutation.mutate()}
              disabled={runMutation.isPending}
              data-testid="run-backtest-button"
            >
              <Play className="mr-2 size-4" />
              {runMutation.isPending ? 'Running...' : 'Run backtest'}
            </Button>
          </>
        )}
      />

      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
          <CardDescription>Backtest parameters</CardDescription>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Start Date</dt>
              <dd className="mt-1 text-sm font-medium">{new Date(config.start_date).toLocaleDateString()}</dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">End Date</dt>
              <dd className="mt-1 text-sm font-medium">{new Date(config.end_date).toLocaleDateString()}</dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Initial Capital</dt>
              <dd className="mt-1 text-sm font-medium">${config.simulation.initial_capital.toLocaleString()}</dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Created</dt>
              <dd className="mt-1 text-sm font-medium">{new Date(config.created_at).toLocaleDateString()}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Run history</CardTitle>
          <CardDescription>Click a run to view its metrics and equity curve</CardDescription>
        </CardHeader>
        <CardContent>
          <BacktestRunHistory
            runs={runs}
            selectedRunId={selectedRun?.id}
            onSelectRun={setSelectedRun}
          />
        </CardContent>
      </Card>

      {selectedRun ? (
        <>
          <BacktestMetricsCard metrics={selectedRun.metrics} />

          <Card>
            <CardHeader>
              <CardTitle>Equity curve</CardTitle>
              <CardDescription>
                Run from {new Date(selectedRun.run_timestamp).toLocaleDateString()}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BacktestEquityChart data={selectedRun.equity_curve ?? []} />
            </CardContent>
          </Card>
        </>
      ) : null}
    </div>
  )
}
