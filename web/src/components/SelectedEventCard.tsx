import { Command, Shield } from 'lucide-react'
import type { EventRow } from '../api'
import { matchedRuleValue, ruleLabel } from '../rules'

type Props = { event?: EventRow }

export function SelectedEventCard({ event }: Props) {
  return <section className="panel detail-card">
    {event ? <>
      <div className="panel-head compact"><div><span className="panel-kicker">Selected event</span><h3>{event.id}</h3></div><button className="icon-button subtle" aria-label="Copy transaction ID" onClick={() => void navigator.clipboard.writeText(event.fullId)}><Command size={15} /></button></div>
      <div className="detail-score"><div><span>Risk score</span><strong>{event.score}</strong></div><div className="detail-amount"><span>Amount</span><b>{event.amount}</b></div><span className={`decision ${event.score >= 80 ? 'review' : 'clear'}`}>{event.score >= 80 ? 'REVIEW' : 'CLEAR'}</span></div>
      <div className="score-breakdown">
        <div><span>Score position</span><b>{event.score} / 100</b></div>
        <div className="score-scale"><i style={{ width: `${Math.min(100, event.score)}%` }} /><span>review 80</span></div>
        <div className="score-facts"><span><small>Triggered rules</small><b>{event.rules.length}</b></span><span><small>Distance to review</small><b>{event.score >= 80 ? `+${event.score - 80}` : `${80 - event.score} below`}</b></span></div>
      </div>
      <div className="transaction-facts">
        <span><small>Country</small><b>{event.country}</b></span>
        <span><small>Currency</small><b>{event.source.transaction?.currency ?? '—'}</b></span>
        <span><small>IP address</small><b>{event.source.transaction?.ip ?? 'snapshot unavailable'}</b></span>
      </div>
      <div className="rule-block"><span>Triggered rules</span><div>{event.rules.length ? event.rules.map((rule) => {
        const matchedValue = matchedRuleValue(rule, event.country)
        return <b key={rule}><span>{ruleLabel(rule)}</span>{matchedValue && <small>Matched value · {matchedValue}</small>}</b>
      }) : <em>No rules matched</em>}</div></div>
      <div className="detail-meta"><span><small>Outbox</small><b><i className={`delivery-dot ${event.deliveryStatus.toLowerCase()}`} />{event.deliveryStatus.replaceAll('_', ' ')}</b></span><span><small>Processed</small><b>{event.time}</b></span></div>
    </> : <div className="empty-inspector"><Shield size={28} /><strong>No event selected</strong><span>Generate an event or wait for the pipeline.</span></div>}
  </section>
}
