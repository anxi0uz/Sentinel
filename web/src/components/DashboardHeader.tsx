import { Search, Shield, Sparkles } from 'lucide-react'

type Props = {
  connected: boolean
  date: string
  loadError: string
  onGenerate: () => void
  onSearchChange: (value: string) => void
  search: string
  timezone: string
}

export function DashboardHeader({ connected, date, loadError, onGenerate, onSearchChange, search, timezone }: Props) {
  return <>
    <header className="topbar">
      <div className="topbar-title">
        <div className="brand-mark"><Shield size={19} strokeWidth={2.2} /></div>
        <strong>Sentinel</strong><span className="title-divider" /><h1>Risk dashboard</h1>
      </div>
      <div className="top-actions">
        <label className="search"><Search size={16} /><input aria-label="Search events" placeholder="Search transaction or user" value={search} onChange={(event) => onSearchChange(event.target.value)} /></label>
        <button className="generate-button" onClick={onGenerate}><Sparkles size={15} /> Generate event</button>
      </div>
    </header>

    <section className="context-bar" aria-label="Pipeline status">
      <div className="live-state"><span className={`live-dot ${connected ? '' : 'offline'}`} /><b>{connected ? 'Pipeline live' : loadError || 'Connecting'}</b><span>/api · polling 3s</span></div>
      <div className="date-chip">{date}<span>{timezone}</span></div>
    </section>
  </>
}
