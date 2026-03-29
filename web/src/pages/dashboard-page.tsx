import { ActiveStrategies } from '@/components/dashboard/active-strategies'
import { ActivityFeed } from '@/components/dashboard/activity-feed'
import { PortfolioSummary } from '@/components/dashboard/portfolio-summary'
import { RiskStatusBar } from '@/components/dashboard/risk-status-bar'

export function DashboardPage() {
  return (
    <div className="space-y-6" data-testid="dashboard-page">
      <PortfolioSummary />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(280px,1fr)]">
        <div className="space-y-6">
          <ActiveStrategies />
          <ActivityFeed />
        </div>

        <div className="space-y-6">
          <RiskStatusBar />
        </div>
      </div>
    </div>
  )
}
