import { type FormEvent, useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { MarketType, Strategy, StrategyUpdateRequest } from '@/lib/api/types'

interface StrategyConfigEditorProps {
  strategy: Strategy
  onSave: (data: StrategyUpdateRequest) => void
  isSaving?: boolean
}

const marketTypes: MarketType[] = ['stock', 'crypto', 'polymarket']

export function StrategyConfigEditor({ strategy, onSave, isSaving }: StrategyConfigEditorProps) {
  const [name, setName] = useState(strategy.name)
  const [description, setDescription] = useState(strategy.description ?? '')
  const [ticker, setTicker] = useState(strategy.ticker)
  const [marketType, setMarketType] = useState<MarketType>(strategy.market_type)
  const [scheduleCron, setScheduleCron] = useState(strategy.schedule_cron ?? '')
  const [isPaper, setIsPaper] = useState(strategy.is_paper)
  const [isActive, setIsActive] = useState(strategy.is_active)
  const [configJson, setConfigJson] = useState(JSON.stringify(strategy.config ?? {}, null, 2))
  const [configError, setConfigError] = useState<string | null>(null)

  useEffect(() => {
    setName(strategy.name)
    setDescription(strategy.description ?? '')
    setTicker(strategy.ticker)
    setMarketType(strategy.market_type)
    setScheduleCron(strategy.schedule_cron ?? '')
    setIsPaper(strategy.is_paper)
    setIsActive(strategy.is_active)
    setConfigJson(JSON.stringify(strategy.config ?? {}, null, 2))
  }, [strategy])

  function handleSubmit(e: FormEvent) {
    e.preventDefault()

    let config: unknown = {}
    try {
      config = JSON.parse(configJson)
      setConfigError(null)
    } catch {
      setConfigError('Invalid JSON')
      return
    }

    onSave({
      name,
      description: description || undefined,
      ticker: ticker.toUpperCase(),
      market_type: marketType,
      schedule_cron: scheduleCron || undefined,
      config,
      is_active: isActive,
      is_paper: isPaper,
    })
  }

  return (
    <Card data-testid="strategy-config-editor">
      <CardHeader>
        <CardTitle>Configuration</CardTitle>
        <CardDescription>Edit strategy settings</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input id="edit-name" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-ticker">Ticker</Label>
              <Input id="edit-ticker" value={ticker} onChange={(e) => setTicker(e.target.value)} required />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-description">Description</Label>
            <Input id="edit-description" value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="edit-market-type">Market type</Label>
              <select
                id="edit-market-type"
                value={marketType}
                onChange={(e) => setMarketType(e.target.value as MarketType)}
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                {marketTypes.map((mt) => (
                  <option key={mt} value={mt}>
                    {mt}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-schedule">Schedule (cron)</Label>
              <Input id="edit-schedule" value={scheduleCron} onChange={(e) => setScheduleCron(e.target.value)} placeholder="0 9 * * 1-5" />
            </div>
          </div>

          <div className="flex gap-6">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={isPaper} onChange={(e) => setIsPaper(e.target.checked)} className="rounded border-input" />
              Paper trading
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={isActive} onChange={(e) => setIsActive(e.target.checked)} className="rounded border-input" />
              Active
            </label>
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-config">Configuration (JSON)</Label>
            <Textarea
              id="edit-config"
              value={configJson}
              onChange={(e) => {
                setConfigJson(e.target.value)
                setConfigError(null)
              }}
              rows={6}
              className="font-mono text-xs"
              data-testid="config-editor-textarea"
            />
            {configError ? (
              <p className="text-xs text-destructive">{configError}</p>
            ) : null}
          </div>

          <div className="flex justify-end">
            <Button type="submit" disabled={isSaving || !name || !ticker}>
              {isSaving ? 'Saving…' : 'Save changes'}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
