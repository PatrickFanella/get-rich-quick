import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Loader2, Play } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'

import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { apiClient } from '@/lib/api/client'

function formatRelativeTime(iso?: string): string {
  if (!iso) return '--'
  const diff = Date.now() - new Date(iso).getTime()
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

export function AutomationDetailPage() {
  const { name } = useParams<{ name: string }>()
  const queryClient = useQueryClient()

  const { data: allJobs, isLoading, isError } = useQuery({
    queryKey: ['automation-status'],
    queryFn: () => apiClient.getAutomationStatus(),
    refetchInterval: 5000,
  })

  const job = allJobs?.find(j => j.name === name)

  const runMutation = useMutation({
    mutationFn: (jobName: string) => apiClient.runAutomationJob(jobName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automation-status'] })
    },
  })

  const enableMutation = useMutation({
    mutationFn: ({ jobName, enabled }: { jobName: string; enabled: boolean }) =>
      apiClient.setAutomationJobEnabled(jobName, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automation-status'] })
    },
  })

  if (isLoading) {
    return (
      <div className="space-y-6" data-testid="automation-detail-loading">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-64 animate-pulse rounded-lg border bg-muted" />
      </div>
    )
  }

  if (isError || !job) {
    return (
      <div className="space-y-4" data-testid="automation-detail-error">
        <Link to="/automation" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" />
          Back to Automation
        </Link>
        <Card>
          <CardContent className="py-8">
            <p className="text-center text-sm text-muted-foreground">
              Unable to load job. It may not exist or the API server is unavailable.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-4" data-testid="automation-detail-page">
      <PageHeader
        title={job.name}
        description={job.description}
        meta={
          <>
            {job.running ? (
              <Badge variant="success">running</Badge>
            ) : (
              <Badge variant="secondary">idle</Badge>
            )}
            {!job.enabled && <Badge variant="warning">disabled</Badge>}
          </>
        }
        actions={
          <>
            <Link
              to="/automation"
              className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-3 py-2 text-sm text-muted-foreground transition-colors hover:border-primary/25 hover:text-foreground"
            >
              <ArrowLeft className="size-4" />
              Back
            </Link>
            <Button
              size="sm"
              variant="outline"
              disabled={job.running || runMutation.isPending}
              onClick={() => runMutation.mutate(job.name)}
            >
              {job.running ? (
                <Loader2 className="mr-1 size-4 animate-spin" />
              ) : (
                <Play className="mr-1 size-4" />
              )}
              {runMutation.isPending ? 'Running...' : 'Run now'}
            </Button>
            <Button
              size="sm"
              variant={job.enabled ? 'outline' : 'secondary'}
              disabled={enableMutation.isPending}
              onClick={() =>
                enableMutation.mutate({ jobName: job.name, enabled: !job.enabled })
              }
            >
              {job.enabled ? 'Disable' : 'Enable'}
            </Button>
          </>
        }
      />

      <Card>
        <CardHeader>
          <CardTitle>Details</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Schedule</dt>
              <dd className="mt-1 text-sm font-medium font-mono">{job.schedule}</dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Status</dt>
              <dd className="mt-1">
                {job.running ? (
                  <Badge variant="success">running</Badge>
                ) : (
                  <Badge variant="secondary">idle</Badge>
                )}
              </dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Enabled</dt>
              <dd className="mt-1 text-sm font-medium">{job.enabled ? 'Yes' : 'No'}</dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Last Run</dt>
              <dd className="mt-1 text-sm font-medium">{formatRelativeTime(job.last_run)}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Run Count</dt>
              <dd className="mt-1 text-2xl font-semibold font-mono">{job.run_count}</dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Error Count</dt>
              <dd className="mt-1 text-2xl font-semibold font-mono">
                {job.error_count > 0 ? (
                  <span className="text-destructive">{job.error_count}</span>
                ) : (
                  job.error_count
                )}
              </dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Last Result</dt>
              <dd className="mt-1">
                {!job.last_result ? (
                  <Badge variant="secondary">--</Badge>
                ) : job.last_result.startsWith('ok') ? (
                  <Badge variant="success">success</Badge>
                ) : job.last_result.startsWith('error') ? (
                  <Badge variant="destructive">failed</Badge>
                ) : (
                  <Badge variant="warning">{job.last_result}</Badge>
                )}
              </dd>
            </div>
            <div>
              <dt className="font-mono text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Last Run Time</dt>
              <dd className="mt-1 text-sm font-medium">
                {job.last_run ? new Date(job.last_run).toLocaleString() : '--'}
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      {job.last_error && (
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">Last Error</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive font-mono whitespace-pre-wrap">
              {job.last_error}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
