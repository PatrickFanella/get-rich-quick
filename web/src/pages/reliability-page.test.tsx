import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { ReliabilityPage } from '@/pages/reliability-page'

const automationHealthResponse = {
  jobs: [],
  healthy: true,
  total_jobs: 0,
  failing_jobs: 0,
  degraded_jobs: 0,
}

const runsResponse = {
  data: [],
  total: 0,
  limit: 50,
  offset: 0,
}

const settingsResponse = {
  llm: {
    default_provider: 'openai',
    deep_think_model: 'gpt-5.2',
    quick_think_model: 'gpt-5-mini',
    providers: {
      openai: {
        api_key_configured: true,
        api_key_last4: '1234',
        base_url: 'https://api.openai.com/v1',
        model: 'gpt-5-mini',
      },
      anthropic: {
        api_key_configured: false,
        model: 'claude-3-7-sonnet-latest',
      },
      google: {
        api_key_configured: false,
        model: 'gemini-2.5-flash',
      },
      openrouter: {
        api_key_configured: false,
        model: 'openai/gpt-4.1-mini',
      },
      xai: {
        api_key_configured: false,
        model: 'grok-3-mini',
      },
      ollama: {
        base_url: 'http://localhost:11434',
        model: 'llama3.2',
      },
    },
  },
  risk: {
    max_position_size_pct: 10,
    max_daily_loss_pct: 2,
    max_drawdown_pct: 8,
    max_open_positions: 6,
    max_total_exposure_pct: 80,
    max_per_market_exposure_pct: 40,
    circuit_breaker_threshold_pct: 5,
    circuit_breaker_cooldown_min: 15,
  },
  system: {
    environment: 'development',
    version: 'v1.2.3',
    current_schema_version: 27,
    required_schema_version: 28,
    schema_status: 'behind',
    uptime_seconds: 5400,
    connected_brokers: [],
  },
}

function Wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchOnWindowFocus: false, refetchOnReconnect: false },
    },
  })

  return <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

describe('ReliabilityPage', () => {
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('renders schema status from settings metadata', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()

      if (url.includes('/api/v1/automation/health')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => automationHealthResponse,
        })
      }

      if (url.includes('/api/v1/settings')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => settingsResponse,
        })
      }

      if (url.includes('/api/v1/runs')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => runsResponse,
        })
      }

      return Promise.reject(new Error(`Unhandled request: ${url}`))
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<ReliabilityPage />, { wrapper: Wrapper })

    const card = await screen.findByTestId('schema-status-card')
    expect(await screen.findByText('27')).toBeInTheDocument()

    expect(within(card).getByText('Schema Status')).toBeInTheDocument()
    expect(within(card).getByText('behind')).toBeInTheDocument()
    expect(within(card).getByText('Current schema')).toBeInTheDocument()
    expect(within(card).getByText('Required schema')).toBeInTheDocument()
    expect(within(card).getByText('27')).toBeInTheDocument()
    expect(within(card).getByText('28')).toBeInTheDocument()
  })
})
