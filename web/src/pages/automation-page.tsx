import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Play, Zap } from 'lucide-react'
import { Link } from 'react-router-dom'

import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { apiClient } from '@/lib/api/client'
import type { JobStatus } from '@/lib/api/types'

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

function resultBadge(result: string) {
  if (!result) return <Badge variant="secondary">--</Badge>
  if (result.startsWith('ok')) return <Badge variant="success">success</Badge>
  if (result.startsWith('error')) return <Badge variant="destructive">failed</Badge>
  return <Badge variant="warning">{result}</Badge>
}

function statusDot(job: JobStatus) {
  if (job.running) return <span className="inline-block size-2 rounded-full bg-emerald-400" />
  if (job.last_error) return <span className="inline-block size-2 rounded-full bg-red-400" />
  return <span className="inline-block size-2 rounded-full bg-zinc-500" />
}

export function AutomationPage() {
  const queryClient = useQueryClient()

  const statusQuery = useQuery({
    queryKey: ['automation-status'],
    queryFn: () => apiClient.getAutomationStatus(),
    refetchInterval: 10_000,
  })

  const runMutation = useMutation({
    mutationFn: (name: string) => apiClient.runAutomationJob(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automation-status'] })
    },
  })

  const enableMutation = useMutation({
    mutationFn: ({ name, enabled }: { name: string; enabled: boolean }) =>
      apiClient.setAutomationJobEnabled(name, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automation-status'] })
    },
  })

  const jobs: JobStatus[] = statusQuery.data ?? []

  return (
    <div className="space-y-4" data-testid="automation-page">
      <PageHeader
        title="Automation"
        description="Scheduled jobs and their execution status."
        meta={<Zap className="size-4 text-muted-foreground" />}
      />

      <Card>
        <CardHeader>
          <CardTitle>Jobs</CardTitle>
        </CardHeader>
        <CardContent>
          {statusQuery.isLoading && (
            <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Loading...
            </div>
          )}

          {statusQuery.isError && (
            <p className="py-4 text-sm text-destructive">
              Failed to load automation status.
            </p>
          )}

          {!statusQuery.isLoading && jobs.length === 0 && (
            <p className="py-4 text-sm text-muted-foreground">
              No automation jobs configured.
            </p>
          )}

          {jobs.length > 0 && (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    <th className="px-2 py-2 w-8" />
                    <th className="px-2 py-2">Name</th>
                    <th className="px-2 py-2">Description</th>
                    <th className="px-2 py-2">Schedule</th>
                    <th className="px-2 py-2">Last Run</th>
                    <th className="px-2 py-2">Last Result</th>
                    <th className="px-2 py-2 text-right">Runs</th>
                    <th className="px-2 py-2 text-right">Errors</th>
                    <th className="px-2 py-2">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {jobs.map((job) => (
                    <>
                      <tr
                        key={job.name}
                        className="border-b border-border/50 hover:bg-accent/30"
                      >
                        <td className="px-2 py-1.5">{statusDot(job)}</td>
                        <td className="px-2 py-1.5 font-mono font-medium">
                          <Link to={`/automation/${job.name}`} className="text-primary hover:underline font-medium">
                            {job.name}
                          </Link>
                        </td>
                        <td className="max-w-48 truncate px-2 py-1.5 text-muted-foreground">
                          {job.description}
                        </td>
                        <td className="px-2 py-1.5 text-muted-foreground">
                          {job.schedule}
                        </td>
                        <td className="px-2 py-1.5 text-xs text-muted-foreground">
                          {formatRelativeTime(job.last_run)}
                        </td>
                        <td className="px-2 py-1.5">{resultBadge(job.last_result)}</td>
                        <td className="px-2 py-1.5 text-right font-mono">
                          {job.run_count}
                        </td>
                        <td className="px-2 py-1.5 text-right font-mono">
                          {job.error_count > 0 ? (
                            <span className="text-destructive">{job.error_count}</span>
                          ) : (
                            job.error_count
                          )}
                        </td>
                        <td className="px-2 py-1.5">
                          <div className="flex items-center gap-1.5">
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={
                                job.running ||
                                runMutation.isPending
                              }
                              onClick={() => runMutation.mutate(job.name)}
                            >
                              {job.running ? (
                                <Loader2 className="mr-1 size-3 animate-spin" />
                              ) : (
                                <Play className="mr-1 size-3" />
                              )}
                              Run
                            </Button>
                            <Button
                              size="sm"
                              variant={job.enabled ? 'outline' : 'secondary'}
                              disabled={enableMutation.isPending}
                              onClick={() =>
                                enableMutation.mutate({
                                  name: job.name,
                                  enabled: !job.enabled,
                                })
                              }
                            >
                              {job.enabled ? 'Disable' : 'Enable'}
                            </Button>
                          </div>
                        </td>
                      </tr>
                    </>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
