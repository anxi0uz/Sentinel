import type { EventRow } from '../api'

type Props = { events: EventRow[] }

export function RiskCadenceCard({ events }: Props) {
  const recentDecisions = events.slice(0, 7).reverse()

  return <section className="panel cadence-card">
    <div className="panel-head compact"><div><span className="panel-kicker">Recent decision profile</span><h3>Last 7 risk scores</h3></div><span className="cadence-summary"><i /> {recentDecisions.length} events</span></div>
    <div className="cadence-copy"><span>Each column is one processed event · oldest → newest</span><b>Review threshold 80</b></div>
    <div className="cadence-chart">
      {recentDecisions.map((event) => <span key={event.fullId}><i><em className={event.severity.toLowerCase()} style={{ height: `${Math.max(8, Math.min(100, event.score))}%` }} /></i><small>{event.time.slice(-5)}</small></span>)}
      {recentDecisions.length === 0 && <div className="table-empty">Waiting for decisions</div>}
    </div>
    <div className="cadence-legend"><span><i className="clear" />Clear</span><span><i className="medium" />Medium</span><span><i className="high" />High</span></div>
  </section>
}
