import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { PortfolioSummary } from '@/components/dashboard/portfolio-summary'

function Wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('PortfolioSummary', () => {
  it('renders portfolio metrics on successful fetch', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ open_positions: 3, unrealized_pnl: 1500.5, realized_pnl: -200.25 }),
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<PortfolioSummary />, { wrapper: Wrapper })

    expect(await screen.findByText('Total P&L')).toBeInTheDocument()
    expect(await screen.findByText('$1,300.25')).toBeInTheDocument()
    expect(screen.getByText('$1,500.50')).toBeInTheDocument()
    expect(screen.getByText('-$200.25')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('shows error state when fetch fails', async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error('Network error'))
    vi.stubGlobal('fetch', fetchMock)

    render(<PortfolioSummary />, { wrapper: Wrapper })

    expect(await screen.findByTestId('portfolio-summary-error')).toBeInTheDocument()
  })
})
