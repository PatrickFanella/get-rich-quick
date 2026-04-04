import { useMutation } from '@tanstack/react-query'
import { AlertTriangle, ChevronDown, ChevronUp, Loader2, Sparkles } from 'lucide-react'
import { useState } from 'react'

import { DiscoveryPipeline } from '@/components/discovery/discovery-pipeline'
import { DiscoveryWinnerCard } from '@/components/discovery/discovery-winner-card'
import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { apiClient } from '@/lib/api/client'
import type { DiscoveryResult, DiscoveryRunRequest } from '@/lib/api/types'

export function DiscoveryPage() {
  const [tickers, setTickers] = useState('AAPL, MSFT, GOOGL, TSLA, NVDA')
  const [marketType, setMarketType] = useState('')
  const [maxWinners, setMaxWinners] = useState(3)
  const [dryRun, setDryRun] = useState(false)
  const [errorsOpen, setErrorsOpen] = useState(false)

  const mutation = useMutation({
    mutationFn: (data: DiscoveryRunRequest) => apiClient.runDiscovery(data),
  })

  const result: DiscoveryResult | undefined = mutation.data

  function handleRun() {
    const parsed = tickers
      .split(',')
      .map((t) => t.trim().toUpperCase())
      .filter(Boolean)
    if (parsed.length === 0) return

    const req: DiscoveryRunRequest = {
      tickers: parsed,
      market_type: marketType || undefined,
      dry_run: dryRun || undefined,
      max_winners: maxWinners,
    }
    mutation.mutate(req)
  }

  return (
    <div className="space-y-4" data-testid="discovery-page">
      <PageHeader
        title="Discovery"
        description="Automatically discover, backtest, and deploy profitable strategies."
      />

      {/* Run Discovery */}
      <Card>
        <CardHeader>
          <CardTitle>Run Discovery</CardTitle>
          <CardDescription>Enter tickers and parameters to start the discovery pipeline.</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-3"
            onSubmit={(e) => {
              e.preventDefault()
              handleRun()
            }}
          >
            <div className="grid gap-3 lg:grid-cols-[1fr_160px_120px_auto]">
              <Input
                placeholder="Tickers (comma-separated, e.g. AAPL, MSFT)"
                value={tickers}
                onChange={(e) => setTickers(e.target.value)}
                aria-label="Tickers"
              />
              <select
                value={marketType}
                onChange={(e) => setMarketType(e.target.value)}
                aria-label="Market type"
                className="flex h-9 w-full rounded-md border border-input bg-card px-3 py-1 text-sm text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
              >
                <option value="">Any market</option>
                <option value="stock">Stock</option>
                <option value="crypto">Crypto</option>
                <option value="options">Options</option>
              </select>
              <Input
                type="number"
                min={1}
                max={20}
                value={maxWinners}
                onChange={(e) => setMaxWinners(Number(e.target.value))}
                aria-label="Max winners"
                placeholder="Max winners"
              />
              <Button type="submit" disabled={mutation.isPending || !tickers.trim()}>
                {mutation.isPending ? (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                ) : (
                  <Sparkles className="mr-2 size-4" />
                )}
                Run Discovery
              </Button>
            </div>
            <label className="inline-flex items-center gap-2 text-sm text-muted-foreground">
              <input
                type="checkbox"
                checked={dryRun}
                onChange={(e) => setDryRun(e.target.checked)}
                className="rounded border-input"
              />
              Dry run
            </label>
          </form>
        </CardContent>
      </Card>

      {/* Loading state */}
      {mutation.isPending && (
        <Card>
          <CardContent className="flex items-center gap-3 py-8">
            <Loader2 className="size-5 animate-spin text-primary" />
            <span className="text-sm text-muted-foreground">Discovering strategies...</span>
          </CardContent>
        </Card>
      )}

      {/* Error state */}
      {mutation.isError && (
        <Card>
          <CardContent className="py-6">
            <p className="text-sm text-destructive">
              Discovery failed: {mutation.error instanceof Error ? mutation.error.message : 'Unknown error'}
            </p>
          </CardContent>
        </Card>
      )}

      {/* Results */}
      {result && (
        <>
          {/* Pipeline visualization */}
          <Card>
            <CardHeader>
              <CardTitle>Pipeline</CardTitle>
            </CardHeader>
            <CardContent>
              <DiscoveryPipeline
                candidates={result.candidates}
                generated={result.generated}
                swept={result.swept}
                validated={result.validated}
                deployed={result.deployed}
              />
            </CardContent>
          </Card>

          {/* Summary */}
          <Card>
            <CardHeader>
              <CardTitle>Summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm">
                Screened <strong>{result.candidates}</strong> candidates
                {' \u2192 '}Generated <strong>{result.generated}</strong> strategies
                {' \u2192 '}Swept <strong>{result.swept}</strong>
                {' \u2192 '}Validated <strong>{result.validated}</strong>
                {' \u2192 '}Deployed <strong>{result.deployed}</strong>
              </p>
              <p className="font-mono text-xs text-muted-foreground">
                Duration: {result.duration.toFixed(1)}s
              </p>

              {/* Errors */}
              {result.errors && result.errors.length > 0 && (
                <div>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1.5 text-sm font-medium text-amber-400 hover:underline"
                    onClick={() => setErrorsOpen((v) => !v)}
                  >
                    <AlertTriangle className="size-3.5" />
                    {result.errors.length} error{result.errors.length > 1 ? 's' : ''}
                    {errorsOpen ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
                  </button>
                  {errorsOpen && (
                    <ul className="mt-2 space-y-1 rounded-md border border-amber-500/20 bg-amber-500/5 p-3">
                      {result.errors.map((err, i) => (
                        <li key={i} className="text-xs text-amber-300">
                          {err}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Winners */}
          {result.winners && result.winners.length > 0 && (
            <div className="space-y-3">
              <h2 className="text-lg font-semibold tracking-tight text-foreground">Winners</h2>
              <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
                {result.winners.map((winner) => (
                  <DiscoveryWinnerCard key={winner.strategy_id} winner={winner} />
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
