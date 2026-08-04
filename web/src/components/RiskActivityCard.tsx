import { ArrowUpRight } from 'lucide-react'
import type { StatsResponse } from '../api'

export type DashboardRange = '24H' | '7D' | '30D'

type Props = {
  alertRate: number
  baselinePoints: string
  chartPoints: string
  connected: boolean
  latestY: number
  onRangeChange: (range: DashboardRange) => void
  publishedRate: number
  range: DashboardRange
  stats: StatsResponse
}

export function RiskActivityCard({ alertRate, baselinePoints, chartPoints, connected, latestY, onRangeChange, publishedRate, range, stats }: Props) {
  return <section className="panel activity-card">
    <div className="panel-head">
      <div><span className="panel-kicker">Processed events</span><h2>{stats.processed.toLocaleString()}</h2></div>
      <div className="range-control">{(['24H', '7D', '30D'] as const).map((item) => <button key={item} onClick={() => onRangeChange(item)} className={range === item ? 'selected' : ''}>{item}</button>)}</div>
    </div>
    <div className="metric-strip">
      <span><small>Flagged</small><strong>{stats.alerts}</strong><em><ArrowUpRight size={11} /> {alertRate}%</em></span>
      <span><small>Average score</small><strong>{stats.average_score.toFixed(1)}</strong><em>recent feed</em></span>
      <span><small>Published</small><strong>{publishedRate}%</strong><em>{connected ? 'streaming' : 'offline'}</em></span>
    </div>
    <div className="risk-chart">
      <div className="chart-y"><span>120</span><span>80</span><span>40</span><span>0</span></div>
      <svg viewBox="0 0 630 160" preserveAspectRatio="none" aria-label="Risk activity chart">
        <defs><linearGradient id="riskArea" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stopColor="#a69cff" stopOpacity=".42" /><stop offset="1" stopColor="#a69cff" stopOpacity="0" /></linearGradient></defs>
        <g className="grid-lines"><line x1="0" x2="630" y1="20" y2="20" /><line x1="0" x2="630" y1="60" y2="60" /><line x1="0" x2="630" y1="100" y2="100" /><line x1="0" x2="630" y1="140" y2="140" /></g>
        <polygon points={`${chartPoints} 630,160 0,160`} fill="url(#riskArea)" />
        <polyline points={baselinePoints} className="safe-line" />
        <polyline points={chartPoints} className="risk-line" />
        <circle cx="630" cy={latestY} r="4" className="chart-point" />
        <circle cx="630" cy={latestY} r="9" className="chart-halo" />
      </svg>
      <div className="chart-x"><span>Oldest</span><span>Recent feed</span><span>Now</span></div>
    </div>
  </section>
}
