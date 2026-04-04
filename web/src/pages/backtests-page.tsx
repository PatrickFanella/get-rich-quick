import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FlaskConical, Play, Plus } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { CreateBacktestDialog } from '@/components/backtests/create-backtest-dialog'
import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { apiClient } from '@/lib/api/client'
import type { BacktestConfigCreateRequest } from '@/lib/api/types'

export function BacktestsPage() {
  const [createOpen, setCreateOpen] = useState(false)
  const queryClient = useQueryClient()

  const { data, isLoading, isError } = useQuery({
    queryKey: ['backtest-configs'],
    queryFn: () => apiClient.listBacktestConfigs(),
    refetchInterval: 30_000,
  })

  const { data: strategiesData } = useQuery({
    queryKey: ['strategies'],
    queryFn: () => apiClient.listStrategies({ limit: 100 }),
  })
  const strategiesById = new Map(
    (strategiesData?.data ?? []).map((s) => [s.id, s]),
  )

  const createMutation = useMutation({
    mutationFn: (req: BacktestConfigCreateRequest) => apiClient.createBacktestConfig(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['backtest-configs'] })
      setCreateOpen(false)
    },
  })

  const runMutation = useMutation({
    mutationFn: (id: string) => apiClient.runBacktestConfig(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['backtest-configs'] })
      queryClient.invalidateQueries({ queryKey: ['backtest-runs'] })
    },
  })

  return (
    <div className="space-y-4" data-testid="backtests-page">
      <PageHeader
        title="Backtests"
        description="Create and run historical backtests against your strategies."
        actions={(
          <Button onClick={() => setCreateOpen(true)} data-testid="create-backtest-button">
            <Plus className="mr-2 size-4" />
            New backtest
          </Button>
        )}
      />

      <Card>
        <CardHeader>
          <CardTitle>All backtest configs</CardTitle>
          <CardDescription>
            {data != null ? `${data.total ?? data.data?.length ?? 0} total configs` : 'Loading...'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2.5" data-testid="backtests-loading">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3 rounded-lg border p-3">
                  <div className="h-4 w-32 animate-pulse rounded bg-muted" />
                  <div className="ml-auto h-5 w-16 animate-pulse rounded-full bg-muted" />
                </div>
              ))}
            </div>
          ) : isError ? (
            <p className="text-sm text-muted-foreground" data-testid="backtests-error">
              Unable to load backtest configs. Start the API server to see live data.
            </p>
          ) : !data?.data?.length ? (
            <div
              className="flex flex-col items-center gap-2 py-8 text-center"
              data-testid="backtests-empty"
            >
              <FlaskConical className="size-8 text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                No backtest configs yet. Create your first to get started.
              </p>
            </div>
          ) : (
            <ul className="space-y-2" data-testid="backtests-list">
              {(data?.data ?? []).map((config) => {
                const strategy = strategiesById.get(config.strategy_id)

                return (
                  <li key={config.id}>
                    <div className="grid gap-3 rounded-lg border border-border p-3 transition-colors hover:bg-accent/45 xl:grid-cols-[minmax(0,1.6fr)_auto] xl:items-center">
                      <div className="flex min-w-0 gap-3">
                        <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
                          <FlaskConical className="size-4" />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <Link
                              to={`/backtests/${config.id}`}
                              className="truncate font-medium hover:text-primary"
                              data-testid={`backtest-link-${config.id}`}
                            >
                              {config.name}
                            </Link>
                            {strategy ? (
                              <Badge variant="secondary">{strategy.name}</Badge>
                            ) : null}
                          </div>
                          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[11px] uppercase tracking-[0.14em] text-muted-foreground">
                            <span>
                              {new Date(config.start_date).toLocaleDateString()} &ndash;{' '}
                              {new Date(config.end_date).toLocaleDateString()}
                            </span>
                            <span>
                              capital ${config.simulation.initial_capital.toLocaleString()}
                            </span>
                            <span>updated {new Date(config.updated_at).toLocaleDateString()}</span>
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-2 xl:justify-end">
                        <Button
                          variant="outline"
                          size="dense"
                          onClick={() => runMutation.mutate(config.id)}
                          disabled={runMutation.isPending}
                          data-testid={`run-backtest-${config.id}`}
                        >
                          <Play className="mr-1 size-3" />
                          Run
                        </Button>
                      </div>
                    </div>
                  </li>
                )
              })}
            </ul>
          )}
        </CardContent>
      </Card>

      <CreateBacktestDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSubmit={(formData) => createMutation.mutate(formData)}
        isSubmitting={createMutation.isPending}
      />
    </div>
  )
}
