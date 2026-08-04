import { ArrowUpRight } from 'lucide-react'
import type { StatsResponse } from '../api'
import { ruleLabel } from '../rules'

const radii = [90, 70, 50, 30]
const colors = ['#a69cff', '#75dcff', '#ee9ae9', '#78a8ff']

type Props = { stats: StatsResponse }

export function SignalDistributionCard({ stats }: Props) {
  const signals = stats.top_rules.slice(0, 4)

  return <section className="panel signals-card">
    <div className="panel-head"><div><span className="panel-kicker">Signal distribution</span><h3>Rule activity</h3></div><ArrowUpRight size={17} /></div>
    <div className="signal-arcs">
      <svg viewBox="0 0 220 130" aria-label="Rule signal distribution">
        {radii.map((radius, index) => {
          const signal = signals[index]
          const percentage = signal ? Math.min(100, Math.round(signal.count / Math.max(1, stats.processed) * 100)) : 0
          const path = `M ${110 - radius} 115 A ${radius} ${radius} 0 0 1 ${110 + radius} 115`
          return <g key={radius}><path d={path} pathLength="100" className="arc-track" /><path d={path} pathLength="100" className="arc-value" stroke={colors[index]} strokeDasharray={`${percentage} 100`} /></g>
        })}
      </svg>
      <div><strong>{stats.alerts}</strong><span>flagged</span></div>
    </div>
    <div className="signal-legend">
      {signals.map((signal, index) => <span key={signal.name}><i className={`signal-color color-${index + 1}`} /><b>{ruleLabel(signal.name)}</b><small>{signal.count}</small></span>)}
    </div>
  </section>
}
