import { type FormEvent, useEffect, useMemo, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { MarketType, Settings, Strategy, StrategyStatus, StrategyUpdateRequest } from '@/lib/api/types'

interface StrategyConfigEditorProps {
  strategy: Strategy
  onSave: (data: StrategyUpdateRequest) => void
  isSaving?: boolean
  settings?: Settings | null
}

const marketTypes: MarketType[] = ['stock', 'crypto', 'polymarket']
const allAnalysts = ['market', 'fundamentals', 'news', 'social']

function resolveStrategyStatus(strategy: Strategy): StrategyStatus {
  if (strategy.status) {
    return strategy.status
  }

  return strategy.is_active ? 'active' : 'inactive'
}

export function StrategyConfigEditor({ strategy, onSave, isSaving, settings }: StrategyConfigEditorProps) {
  const [name, setName] = useState(strategy.name)
  const [description, setDescription] = useState(strategy.description ?? '')
  const [ticker, setTicker] = useState(strategy.ticker)
  const [marketType, setMarketType] = useState<MarketType>(strategy.market_type)
  const [scheduleCron, setScheduleCron] = useState(strategy.schedule_cron ?? '')
  const [isPaper, setIsPaper] = useState(strategy.is_paper)
  const [isActive, setIsActive] = useState(resolveStrategyStatus(strategy) === 'active')
  const [configJson, setConfigJson] = useState(JSON.stringify(strategy.config ?? {}, null, 2))
  const [configError, setConfigError] = useState<string | null>(null)
  const [deepThinkProvider, setDeepThinkProvider] = useState('')
  const [deepThinkModel, setDeepThinkModel] = useState('')
  const [quickThinkProvider, setQuickThinkProvider] = useState('')
  const [quickThinkModel, setQuickThinkModel] = useState('')
  const [researchDebateRounds, setResearchDebateRounds] = useState('')
  const [riskDebateRounds, setRiskDebateRounds] = useState('')
  const [phaseTimeout, setPhaseTimeout] = useState('')
  const [pipelineTimeout, setPipelineTimeout] = useState('')
  const [maxPositionSizePct, setMaxPositionSizePct] = useState('')
  const [stopLossAtrMultiplier, setStopLossAtrMultiplier] = useState('')
  const [takeProfitAtrMultiplier, setTakeProfitAtrMultiplier] = useState('')
  const [minConfidenceThreshold, setMinConfidenceThreshold] = useState('')
  const [selectedAnalysts, setSelectedAnalysts] = useState<string[]>(allAnalysts)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [promptOverrides, setPromptOverrides] = useState('{}')

  useEffect(() => {
    const cfg = (strategy.config ?? {}) as Record<string, unknown>
    const llm = (cfg.llm ?? {}) as Record<string, unknown>
    const pipeline = (cfg.pipeline ?? {}) as Record<string, unknown>
    const risk = (cfg.risk ?? {}) as Record<string, unknown>
    const analysts = Array.isArray(cfg.analysts) ? (cfg.analysts as string[]) : allAnalysts

    setName(strategy.name)
    setDescription(strategy.description ?? '')
    setTicker(strategy.ticker)
    setMarketType(strategy.market_type)
    setScheduleCron(strategy.schedule_cron ?? '')
    setIsPaper(strategy.is_paper)
    setIsActive(resolveStrategyStatus(strategy) === 'active')
    setConfigJson(JSON.stringify(strategy.config ?? {}, null, 2))
    setConfigError(null)

    setDeepThinkProvider((llm.deep_think_provider as string) ?? '')
    setDeepThinkModel((llm.deep_think_model as string) ?? '')
    setQuickThinkProvider((llm.quick_think_provider as string) ?? '')
    setQuickThinkModel((llm.quick_think_model as string) ?? '')

    setResearchDebateRounds(
      pipeline.research_debate_rounds == null ? '' : String(pipeline.research_debate_rounds),
    )
    setRiskDebateRounds(
      pipeline.risk_debate_rounds == null ? '' : String(pipeline.risk_debate_rounds),
    )
    setPhaseTimeout((pipeline.phase_timeout as string) ?? '')
    setPipelineTimeout((pipeline.pipeline_timeout as string) ?? '')

    setMaxPositionSizePct(
      risk.max_position_size_pct == null ? '' : String(risk.max_position_size_pct),
    )
    setStopLossAtrMultiplier(
      risk.stop_loss_atr_multiplier == null ? '' : String(risk.stop_loss_atr_multiplier),
    )
    setTakeProfitAtrMultiplier(
      risk.take_profit_atr_multiplier == null ? '' : String(risk.take_profit_atr_multiplier),
    )
    setMinConfidenceThreshold(
      risk.min_confidence_threshold == null ? '' : String(risk.min_confidence_threshold),
    )

    setSelectedAnalysts(analysts.length > 0 ? analysts : allAnalysts)
    setPromptOverrides(JSON.stringify((cfg.prompt_overrides ?? {}) as Record<string, string>, null, 2))
    setShowAdvanced(false)
  }, [strategy])

  const providerOptions = useMemo(
    () => (settings?.llm?.providers ? Object.keys(settings.llm.providers) : []),
    [settings?.llm?.providers],
  )

  function toggleAnalyst(analyst: string, checked: boolean) {
    setSelectedAnalysts((prev) => {
      if (checked) {
        return prev.includes(analyst) ? prev : [...prev, analyst]
      }
      return prev.filter((value) => value !== analyst)
    })
  }

  function numberValue(value: string) {
    if (!value.trim()) {
      return undefined
    }

    return Number(value)
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()

    let config: Record<string, unknown> = {}
    try {
      config = JSON.parse(configJson) as Record<string, unknown>
    } catch {
      setConfigError('Invalid JSON')
      return
    }

    if (selectedAnalysts.length === 0) {
      setConfigError('Select at least one analyst')
      return
    }

    let parsedPromptOverrides: Record<string, string> = {}
    try {
      parsedPromptOverrides = JSON.parse(promptOverrides || '{}') as Record<string, string>
    } catch {
      setConfigError('Prompt overrides must be valid JSON')
      return
    }

    setConfigError(null)

    const llmConfig: Record<string, string> = {}
    if (deepThinkProvider) llmConfig.deep_think_provider = deepThinkProvider
    if (deepThinkModel) llmConfig.deep_think_model = deepThinkModel
    if (quickThinkProvider) llmConfig.quick_think_provider = quickThinkProvider
    if (quickThinkModel) llmConfig.quick_think_model = quickThinkModel
    if (Object.keys(llmConfig).length > 0) {
      config.llm = { ...((config.llm as Record<string, unknown>) ?? {}), ...llmConfig }
    }

    const pipelineConfig: Record<string, unknown> = {}
    if (researchDebateRounds) pipelineConfig.research_debate_rounds = numberValue(researchDebateRounds)
    if (riskDebateRounds) pipelineConfig.risk_debate_rounds = numberValue(riskDebateRounds)
    if (phaseTimeout.trim()) pipelineConfig.phase_timeout = phaseTimeout.trim()
    if (pipelineTimeout.trim()) pipelineConfig.pipeline_timeout = pipelineTimeout.trim()
    if (Object.keys(pipelineConfig).length > 0) {
      config.pipeline = { ...((config.pipeline as Record<string, unknown>) ?? {}), ...pipelineConfig }
    }

    const riskConfig: Record<string, unknown> = {}
    if (maxPositionSizePct) riskConfig.max_position_size_pct = numberValue(maxPositionSizePct)
    if (stopLossAtrMultiplier) riskConfig.stop_loss_atr_multiplier = numberValue(stopLossAtrMultiplier)
    if (takeProfitAtrMultiplier) riskConfig.take_profit_atr_multiplier = numberValue(takeProfitAtrMultiplier)
    if (minConfidenceThreshold) riskConfig.min_confidence_threshold = numberValue(minConfidenceThreshold)
    if (Object.keys(riskConfig).length > 0) {
      config.risk = { ...((config.risk as Record<string, unknown>) ?? {}), ...riskConfig }
    }

    config.analysts = selectedAnalysts

    if (Object.keys(parsedPromptOverrides).length > 0) {
      config.prompt_overrides = parsedPromptOverrides
    }

    const currentStatus = resolveStrategyStatus(strategy)
    const nextStatus: StrategyStatus = isActive ? 'active' : currentStatus === 'paused' ? 'paused' : 'inactive'

    onSave({
      name,
      description: description || undefined,
      ticker: ticker.toUpperCase(),
      market_type: marketType,
      schedule_cron: scheduleCron || undefined,
      config,
      status: nextStatus,
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

          <div className="space-y-4 rounded-lg border p-4">
            <h4 className="text-sm font-medium">LLM Configuration</h4>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="deep-think-provider">Deep Think Provider</Label>
                <select
                  id="deep-think-provider"
                  value={deepThinkProvider}
                  onChange={(e) => setDeepThinkProvider(e.target.value)}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                >
                  <option value="">Use global default</option>
                  {providerOptions.map((provider) => (
                    <option key={provider} value={provider}>{provider}</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="deep-think-model">Deep Think Model</Label>
                <Input
                  id="deep-think-model"
                  value={deepThinkModel}
                  onChange={(e) => setDeepThinkModel(e.target.value)}
                  placeholder={settings?.llm?.deep_think_model ?? 'Global default'}
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="quick-think-provider">Quick Think Provider</Label>
                <select
                  id="quick-think-provider"
                  value={quickThinkProvider}
                  onChange={(e) => setQuickThinkProvider(e.target.value)}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                >
                  <option value="">Use global default</option>
                  {providerOptions.map((provider) => (
                    <option key={provider} value={provider}>{provider}</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="quick-think-model">Quick Think Model</Label>
                <Input
                  id="quick-think-model"
                  value={quickThinkModel}
                  onChange={(e) => setQuickThinkModel(e.target.value)}
                  placeholder={settings?.llm?.quick_think_model ?? 'Global default'}
                />
              </div>
            </div>
          </div>

          <div className="space-y-4 rounded-lg border p-4">
            <h4 className="text-sm font-medium">Pipeline</h4>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="research-debate-rounds">Research Debate Rounds</Label>
                <Input id="research-debate-rounds" type="number" min={1} max={10} value={researchDebateRounds} onChange={(e) => setResearchDebateRounds(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="risk-debate-rounds">Risk Debate Rounds</Label>
                <Input id="risk-debate-rounds" type="number" min={1} max={10} value={riskDebateRounds} onChange={(e) => setRiskDebateRounds(e.target.value)} />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="phase-timeout">Phase Timeout</Label>
                <Input id="phase-timeout" value={phaseTimeout} onChange={(e) => setPhaseTimeout(e.target.value)} placeholder="2m" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pipeline-timeout">Pipeline Timeout</Label>
                <Input id="pipeline-timeout" value={pipelineTimeout} onChange={(e) => setPipelineTimeout(e.target.value)} placeholder="10m" />
              </div>
            </div>
          </div>

          <div className="space-y-4 rounded-lg border p-4">
            <h4 className="text-sm font-medium">Risk</h4>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="max-position-size-pct">Max Position Size %</Label>
                <Input id="max-position-size-pct" type="number" step="0.01" min={0.01} max={1} value={maxPositionSizePct} onChange={(e) => setMaxPositionSizePct(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="stop-loss-atr-multiplier">Stop Loss ATR Multiplier</Label>
                <Input id="stop-loss-atr-multiplier" type="number" step="0.1" value={stopLossAtrMultiplier} onChange={(e) => setStopLossAtrMultiplier(e.target.value)} />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="take-profit-atr-multiplier">Take Profit ATR Multiplier</Label>
                <Input id="take-profit-atr-multiplier" type="number" step="0.1" value={takeProfitAtrMultiplier} onChange={(e) => setTakeProfitAtrMultiplier(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="min-confidence-threshold">Min Confidence Threshold</Label>
                <Input id="min-confidence-threshold" type="number" step="0.01" min={0} max={1} value={minConfidenceThreshold} onChange={(e) => setMinConfidenceThreshold(e.target.value)} />
              </div>
            </div>
          </div>

          <div className="space-y-4 rounded-lg border p-4">
            <h4 className="text-sm font-medium">Analysts</h4>
            <div className="grid gap-3 sm:grid-cols-2">
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={selectedAnalysts.includes('market')} onChange={(e) => toggleAnalyst('market', e.target.checked)} className="rounded border-input" />
                Market Analyst
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={selectedAnalysts.includes('fundamentals')} onChange={(e) => toggleAnalyst('fundamentals', e.target.checked)} className="rounded border-input" />
                Fundamentals Analyst
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={selectedAnalysts.includes('news')} onChange={(e) => toggleAnalyst('news', e.target.checked)} className="rounded border-input" />
                News Analyst
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={selectedAnalysts.includes('social')} onChange={(e) => toggleAnalyst('social', e.target.checked)} className="rounded border-input" />
                Social Media Analyst
              </label>
            </div>
          </div>

          <div className="space-y-3 rounded-lg border p-4">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-medium">Advanced</h4>
              <Button type="button" variant="outline" size="sm" onClick={() => setShowAdvanced((prev) => !prev)}>
                {showAdvanced ? 'Hide' : 'Show'}
              </Button>
            </div>
            {showAdvanced ? (
              <div className="space-y-2">
                <Label htmlFor="prompt-overrides">Prompt Overrides (JSON)</Label>
                <Textarea
                  id="prompt-overrides"
                  value={promptOverrides}
                  onChange={(e) => setPromptOverrides(e.target.value)}
                  rows={6}
                  className="font-mono text-xs"
                />
              </div>
            ) : null}
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
