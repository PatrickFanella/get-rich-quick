import { useQuery } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { apiClient } from '@/lib/api/client'
import type { BacktestConfigCreateRequest } from '@/lib/api/types'

interface CreateBacktestDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (data: BacktestConfigCreateRequest) => void
  isSubmitting?: boolean
}

const denseSelectClassName =
  'flex h-9 w-full rounded-md border border-input bg-card px-3 py-1 text-sm text-foreground transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring'

export function CreateBacktestDialog({
  open,
  onOpenChange,
  onSubmit,
  isSubmitting,
}: CreateBacktestDialogProps) {
  const [name, setName] = useState('')
  const [strategyId, setStrategyId] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [initialCapital, setInitialCapital] = useState('100000')

  const { data: strategiesData } = useQuery({
    queryKey: ['strategies'],
    queryFn: () => apiClient.listStrategies({ limit: 100 }),
    enabled: open,
  })
  const strategies = strategiesData?.data ?? []

  function resetForm() {
    setName('')
    setStrategyId('')
    setStartDate('')
    setEndDate('')
    setInitialCapital('100000')
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) {
      resetForm()
    }
    onOpenChange(nextOpen)
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()

    onSubmit({
      name,
      strategy_id: strategyId,
      start_date: startDate ? `${startDate}T00:00:00Z` : '',
      end_date: endDate ? `${endDate}T00:00:00Z` : '',
      simulation: {
        initial_capital: Number(initialCapital),
      },
    })
  }

  const isValid = name && strategyId && startDate && endDate && Number(initialCapital) > 0

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent data-testid="create-backtest-dialog">
        <DialogHeader>
          <DialogTitle>Create backtest</DialogTitle>
          <DialogDescription>
            Configure a new backtest. Required fields are marked with *.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2 rounded-lg border border-border bg-background p-4">
            <Label htmlFor="backtest-name">Name *</Label>
            <Input
              id="backtest-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Q1 2025 Backtest"
              required
              data-testid="backtest-name-input"
            />
          </div>

          <div className="space-y-2 rounded-lg border border-border bg-background p-4">
            <Label htmlFor="backtest-strategy">Strategy *</Label>
            <select
              id="backtest-strategy"
              value={strategyId}
              onChange={(e) => setStrategyId(e.target.value)}
              className={denseSelectClassName}
              required
              data-testid="backtest-strategy-select"
            >
              <option value="">Select a strategy</option>
              {strategies.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2 rounded-lg border border-border bg-background p-4">
              <Label htmlFor="backtest-start-date">Start Date *</Label>
              <Input
                id="backtest-start-date"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                required
                data-testid="backtest-start-date-input"
              />
            </div>
            <div className="space-y-2 rounded-lg border border-border bg-background p-4">
              <Label htmlFor="backtest-end-date">End Date *</Label>
              <Input
                id="backtest-end-date"
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                required
                data-testid="backtest-end-date-input"
              />
            </div>
          </div>

          <div className="space-y-2 rounded-lg border border-border bg-background p-4">
            <Label htmlFor="backtest-initial-capital">Initial Capital *</Label>
            <Input
              id="backtest-initial-capital"
              type="number"
              min={1}
              value={initialCapital}
              onChange={(e) => setInitialCapital(e.target.value)}
              required
              data-testid="backtest-initial-capital-input"
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              size="dense"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" size="dense" disabled={isSubmitting || !isValid} data-testid="create-backtest-submit">
              {isSubmitting ? 'Creating...' : 'Create backtest'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
