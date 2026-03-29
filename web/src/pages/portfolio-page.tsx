import { PortfolioSummary } from '@/components/dashboard/portfolio-summary'
import { PortfolioChart } from '@/components/portfolio/portfolio-chart'
import { PositionsTable } from '@/components/portfolio/positions-table'
import { TradeHistory } from '@/components/portfolio/trade-history'

export function PortfolioPage() {
  return (
    <div className="space-y-6" data-testid="portfolio-page">
      <PortfolioSummary />

      <PortfolioChart />

      <div className="grid gap-6 lg:grid-cols-2">
        <PositionsTable />
        <TradeHistory />
      </div>
    </div>
  )
}
