import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, CheckCircle2, Power, Shield, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { apiClient } from '@/lib/api/client'
import type { EngineStatus, RiskStatus } from '@/lib/api/types'
import { cn } from '@/lib/utils'

function riskStatusConfig(status: RiskStatus) {
  switch (status) {
    case 'normal':
      return {
        icon: CheckCircle2,
        label: 'Normal',
        variant: 'success' as const,
        color: 'text-emerald-600',
      }
    case 'warning':
      return {
        icon: AlertTriangle,
        label: 'Warning',
        variant: 'warning' as const,
        color: 'text-amber-600',
      }
    case 'breached':
      return {
        icon: XCircle,
        label: 'Breached',
        variant: 'destructive' as const,
        color: 'text-red-600',
      }
  }
}

function CircuitBreakerDisplay({ status }: { status: EngineStatus }) {
  const { circuit_breaker: cb } = status

  const stateLabels: Record<string, string> = {
    open: 'Open',
    tripped: 'Tripped',
    cooldown: 'Cooldown',
  }

  const stateVariants: Record<string, 'success' | 'destructive' | 'warning'> = {
    open: 'success',
    tripped: 'destructive',
    cooldown: 'warning',
  }

  return (
    <div className="rounded-lg border p-3">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium">Circuit breaker</p>
        <Badge variant={stateVariants[cb.state] ?? 'secondary'}>{stateLabels[cb.state] ?? cb.state}</Badge>
      </div>
      {cb.reason ? <p className="mt-1 text-xs text-muted-foreground">{cb.reason}</p> : null}
    </div>
  )
}

function PositionLimitsDisplay({ status }: { status: EngineStatus }) {
  const { position_limits: limits } = status

  const items = [
    { label: 'Per position', value: `${limits.max_per_position_pct}%` },
    { label: 'Total exposure', value: `${limits.max_total_pct}%` },
    { label: 'Max concurrent', value: String(limits.max_concurrent) },
    { label: 'Per market', value: `${limits.max_per_market_pct}%` },
  ]

  return (
    <div className="rounded-lg border p-3">
      <p className="mb-2 text-sm font-medium">Position limits</p>
      <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
        {items.map(({ label, value }) => (
          <div key={label} className="flex justify-between">
            <span className="text-muted-foreground">{label}</span>
            <span className="font-medium">{value}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

export function RiskStatusBar() {
  const queryClient = useQueryClient()

  const { data, isLoading, isError } = useQuery({
    queryKey: ['risk', 'status'],
    queryFn: () => apiClient.getRiskStatus(),
    refetchInterval: 15_000,
  })

  const killSwitchMutation = useMutation({
    mutationFn: (active: boolean) =>
      apiClient.toggleKillSwitch({
        active,
        reason: active ? 'Activated from dashboard' : 'Deactivated from dashboard',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['risk', 'status'] })
    },
  })

  if (isLoading) {
    return (
      <Card data-testid="risk-status-loading">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <div className="h-4 w-32 animate-pulse rounded bg-muted" />
          <div className="size-4 animate-pulse rounded bg-muted" />
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="h-16 animate-pulse rounded bg-muted" />
            <div className="h-16 animate-pulse rounded bg-muted" />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (isError || !data) {
    return (
      <Card data-testid="risk-status-error">
        <CardHeader>
          <CardTitle>Risk status</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Unable to load risk status. Start the API server to see live data.
          </p>
        </CardContent>
      </Card>
    )
  }

  const config = riskStatusConfig(data.risk_status)
  const StatusIcon = config.icon

  return (
    <Card data-testid="risk-status">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Shield className="size-5" />
              Risk status
            </CardTitle>
            <CardDescription>Engine health and risk controls</CardDescription>
          </div>
          <Badge variant={config.variant} className="gap-1">
            <StatusIcon className="size-3" />
            {config.label}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <CircuitBreakerDisplay status={data} />
        <PositionLimitsDisplay status={data} />

        <div className="rounded-lg border p-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Kill switch</p>
              <p className="text-xs text-muted-foreground">
                {data.kill_switch.active
                  ? data.kill_switch.reason ?? 'All trading halted'
                  : 'Trading is enabled'}
              </p>
            </div>
            <Button
              variant={data.kill_switch.active ? 'outline' : 'default'}
              size="sm"
              disabled={killSwitchMutation.isPending}
              onClick={() => killSwitchMutation.mutate(!data.kill_switch.active)}
              data-testid="kill-switch-toggle"
            >
              <Power className={cn('size-4', data.kill_switch.active && 'text-emerald-600')} />
              {data.kill_switch.active ? 'Deactivate' : 'Activate'}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
