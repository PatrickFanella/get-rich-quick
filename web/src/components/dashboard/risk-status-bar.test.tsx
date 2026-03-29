import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { RiskStatusBar } from '@/components/dashboard/risk-status-bar'

function Wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

afterEach(() => {
  vi.unstubAllGlobals()
})

const engineStatus = {
  risk_status: 'normal',
  circuit_breaker: { state: 'open', reason: '' },
  kill_switch: { active: false, reason: '', mechanisms: [], activated_at: null },
  position_limits: {
    max_per_position_pct: 10,
    max_total_pct: 80,
    max_concurrent: 5,
    max_per_market_pct: 40,
  },
  updated_at: '2025-01-01T00:00:00Z',
}

describe('RiskStatusBar', () => {
  it('renders risk status on successful fetch', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => engineStatus,
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<RiskStatusBar />, { wrapper: Wrapper })

    expect(await screen.findByText('Normal')).toBeInTheDocument()
    expect(screen.getByText('Circuit breaker')).toBeInTheDocument()
    expect(screen.getByText('Open')).toBeInTheDocument()
    expect(screen.getByText('Kill switch')).toBeInTheDocument()
    expect(screen.getByText('Activate')).toBeInTheDocument()
    expect(screen.getByText('Position limits')).toBeInTheDocument()
    expect(screen.getByText('10%')).toBeInTheDocument()
  })

  it('renders kill switch deactivate when active', async () => {
    const activeStatus = {
      ...engineStatus,
      risk_status: 'warning',
      kill_switch: { active: true, reason: 'Manual stop', mechanisms: ['api_toggle'], activated_at: '2025-01-01T00:00:00Z' },
    }

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => activeStatus,
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<RiskStatusBar />, { wrapper: Wrapper })

    expect(await screen.findByText('Warning')).toBeInTheDocument()
    expect(screen.getByText('Deactivate')).toBeInTheDocument()
    expect(screen.getByText('Manual stop')).toBeInTheDocument()
  })

  it('shows error state when fetch fails', async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error('Network error'))
    vi.stubGlobal('fetch', fetchMock)

    render(<RiskStatusBar />, { wrapper: Wrapper })

    expect(await screen.findByTestId('risk-status-error')).toBeInTheDocument()
  })
})
