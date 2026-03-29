import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Play, Trash2 } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { StrategyConfigEditor } from '@/components/strategies/strategy-config-editor'
import { StrategyRunHistory } from '@/components/strategies/strategy-run-history'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { apiClient } from '@/lib/api/client'
import type { StrategyUpdateRequest } from '@/lib/api/types'

export function StrategyDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: strategy, isLoading, isError } = useQuery({
    queryKey: ['strategy', id],
    queryFn: () => apiClient.getStrategy(id!),
    enabled: !!id,
  })

  const updateMutation = useMutation({
    mutationFn: (data: StrategyUpdateRequest) => apiClient.updateStrategy(id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['strategy', id] })
      queryClient.invalidateQueries({ queryKey: ['strategies'] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => apiClient.deleteStrategy(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['strategies'] })
      navigate('/strategies')
    },
  })

  const runMutation = useMutation({
    mutationFn: () => apiClient.runStrategy(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['runs', { strategy_id: id }] })
    },
  })

  if (isLoading) {
    return (
      <div className="space-y-6" data-testid="strategy-detail-loading">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-64 animate-pulse rounded-lg border bg-muted" />
      </div>
    )
  }

  if (isError || !strategy) {
    return (
      <div className="space-y-4" data-testid="strategy-detail-error">
        <Link to="/strategies" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" />
          Back to strategies
        </Link>
        <Card>
          <CardContent className="py-8">
            <p className="text-center text-sm text-muted-foreground">
              Unable to load strategy. It may have been deleted or the API server is unavailable.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6" data-testid="strategy-detail-page">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <Link
            to="/strategies"
            className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="size-4" />
            Back to strategies
          </Link>
          <h2 className="text-2xl font-semibold tracking-tight">{strategy.name}</h2>
          {strategy.description ? (
            <p className="text-sm text-muted-foreground">{strategy.description}</p>
          ) : null}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => runMutation.mutate()}
            disabled={runMutation.isPending}
            data-testid="run-strategy-button"
          >
            <Play className="mr-2 size-4" />
            {runMutation.isPending ? 'Running…' : 'Run now'}
          </Button>
          <Button
            variant="outline"
            onClick={() => deleteMutation.mutate()}
            disabled={deleteMutation.isPending}
            data-testid="delete-strategy-button"
          >
            <Trash2 className="mr-2 size-4" />
            Delete
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Overview</CardTitle>
          <CardDescription>Strategy summary and current state</CardDescription>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <dt className="text-xs text-muted-foreground">Ticker</dt>
              <dd className="text-sm font-medium">{strategy.ticker}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted-foreground">Market type</dt>
              <dd>
                <Badge variant={strategy.market_type === 'stock' ? 'default' : strategy.market_type === 'crypto' ? 'secondary' : 'outline'}>
                  {strategy.market_type}
                </Badge>
              </dd>
            </div>
            <div>
              <dt className="text-xs text-muted-foreground">Status</dt>
              <dd className="flex items-center gap-2">
                {strategy.is_active ? (
                  <Badge variant="success">active</Badge>
                ) : (
                  <Badge variant="secondary">inactive</Badge>
                )}
                {strategy.is_paper ? <Badge variant="warning">paper</Badge> : null}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-muted-foreground">Schedule</dt>
              <dd className="text-sm font-medium">
                {strategy.schedule_cron || 'Manual only'}
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <div className="grid gap-6 lg:grid-cols-2">
        <StrategyRunHistory strategyId={strategy.id} />
        <StrategyConfigEditor
          strategy={strategy}
          onSave={(data) => updateMutation.mutate(data)}
          isSaving={updateMutation.isPending}
        />
      </div>
    </div>
  )
}
