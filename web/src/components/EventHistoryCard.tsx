import { ChevronRight, Shield } from 'lucide-react'
import type { EventRow } from '../api'

export type EventFilter = 'all' | 'clear' | 'flagged'

type Props = {
  connected: boolean
  events: EventRow[]
  filter: EventFilter
  onFilterChange: (filter: EventFilter) => void
  onSelect: (event: EventRow) => void
  pendingID: string
  selectedID?: string
}

export function EventHistoryCard({ connected, events, filter, onFilterChange, onSelect, pendingID, selectedID }: Props) {
  return <section className="panel history-card" id="events">
    <div className="panel-head compact">
      <div><span className="panel-kicker">Live history</span><h3>Latest decisions</h3></div><span className="history-count">{events.length}</span>
    </div>
    <div className="history-filters">
      {(['all', 'clear', 'flagged'] as const).map((item) => <button type="button" className={filter === item ? 'selected' : ''} key={item} onClick={() => onFilterChange(item)}>{item}</button>)}
    </div>
    <div className="history-list">
      {pendingID && <div className="pending-event"><span className="live-dot" /> Processing {pendingID.slice(0, 8)}…</div>}
      {events.map((event) => <button className={`history-row ${selectedID === event.fullId ? 'selected-row' : ''}`} key={event.fullId} onClick={() => onSelect(event)}>
        <span className={`history-symbol ${event.severity.toLowerCase()}`}><Shield size={16} /></span>
        <span className="history-id"><b>{event.id}</b><small>{event.country} · {event.time}</small></span>
        <span className="history-result"><b>{event.amount}</b><small className={event.severity.toLowerCase()}>{event.severity} · {event.score}</small></span>
        <ChevronRight size={14} />
      </button>)}
      {!pendingID && events.length === 0 && <div className="table-empty">{connected ? 'No processed events in this range' : 'Waiting for Sentinel API'}</div>}
    </div>
  </section>
}
