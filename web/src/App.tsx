import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  chartPolyline,
  getStats,
  getTransaction,
  getTransactions,
  submitTransaction,
  toEventRow,
  type EventRow,
  type StatsResponse,
  type SubmitTransaction,
  type TransactionEvent,
} from './api'
import { DashboardHeader } from './components/DashboardHeader'
import { type EventFilter, EventHistoryCard } from './components/EventHistoryCard'
import { GenerateEventModal } from './components/GenerateEventModal'
import { RiskActivityCard, type DashboardRange } from './components/RiskActivityCard'
import { RiskCadenceCard } from './components/RiskCadenceCard'
import { SelectedEventCard } from './components/SelectedEventCard'
import { SignalDistributionCard } from './components/SignalDistributionCard'

const emptyStats: StatsResponse = {
  alerts: 0,
  average_score: 0,
  by_severity: { medium: 0, high: 0, critical: 0 },
  processed: 0,
  top_rules: [],
}

function App() {
  const [apiEvents, setAPIEvents] = useState<TransactionEvent[]>([])
  const [stats, setStats] = useState<StatsResponse>(emptyStats)
  const [selectedID, setSelectedID] = useState<string>()
  const [range, setRange] = useState<DashboardRange>('24H')
  const [eventFilter, setEventFilter] = useState<EventFilter>('all')
  const [search, setSearch] = useState('')
  const [connected, setConnected] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [submitOpen, setSubmitOpen] = useState(false)
  const [pendingID, setPendingID] = useState('')

  const date = useMemo(
    () => new Intl.DateTimeFormat('en', { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date()),
    [],
  )
  const timezone = useMemo(() => {
    const offset = -new Date().getTimezoneOffset()
    if (offset === 0) return 'UTC'
    const sign = offset > 0 ? '+' : '-'
    const hours = Math.floor(Math.abs(offset) / 60)
    const minutes = Math.abs(offset) % 60
    return `UTC${sign}${hours}${minutes ? `:${String(minutes).padStart(2, '0')}` : ''}`
  }, [])

  const loadDashboard = useCallback(async (signal?: AbortSignal) => {
    try {
      const [transactions, nextStats] = await Promise.all([getTransactions(signal), getStats(signal)])
      setAPIEvents(transactions.items)
      setStats(nextStats)
      setConnected(true)
      setLoadError('')
      setSelectedID((current) => current && transactions.items.some((event) => event.transaction_id === current)
        ? current
        : transactions.items[0]?.transaction_id)
    } catch (error) {
      if (signal?.aborted) return
      setConnected(false)
      setLoadError(error instanceof Error ? error.message : 'Cannot reach Sentinel API')
    }
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    void loadDashboard(controller.signal)
    const polling = window.setInterval(() => void loadDashboard(), 3000)
    return () => {
      controller.abort()
      window.clearInterval(polling)
    }
  }, [loadDashboard])

  useEffect(() => {
    if (pendingID && apiEvents.some((event) => event.transaction_id === pendingID)) {
      setSelectedID(pendingID)
      setPendingID('')
    }
  }, [apiEvents, pendingID])

  const rangeHours = range === '24H' ? 24 : range === '7D' ? 168 : 720
  const visibleAPIEvents = useMemo(() => {
    const cutoff = Date.now() - rangeHours * 60 * 60 * 1000
    return apiEvents.filter((event) => new Date(event.processed_at).getTime() >= cutoff)
  }, [apiEvents, rangeHours])
  const rows = useMemo(() => visibleAPIEvents.map(toEventRow), [visibleAPIEvents])
  const normalizedSearch = search.trim().toLowerCase()
  const filteredRows = useMemo(() => rows.filter((event) => {
    const matchesSearch = !normalizedSearch
      || event.fullId.toLowerCase().includes(normalizedSearch)
      || event.user.toLowerCase().includes(normalizedSearch)
    const matchesFilter = eventFilter === 'all'
      || (eventFilter === 'clear' && event.severity === 'CLEAR')
      || (eventFilter === 'flagged' && event.severity !== 'CLEAR')
    return matchesSearch && matchesFilter
  }), [rows, normalizedSearch, eventFilter])

  const selected = filteredRows.find((event) => event.fullId === selectedID) ?? filteredRows[0] ?? rows[0]
  const chartPoints = chartPolyline(visibleAPIEvents)
  const baselineY = Math.round(150 - Math.max(0, Math.min(120, stats.average_score)) * 1.15)
  const latestY = Math.round(150 - Math.max(0, Math.min(120, apiEvents[0]?.score ?? 0)) * 1.15)
  const recentAlerts = rows.filter((event) => event.severity !== 'CLEAR')
  const publishedAlerts = recentAlerts.filter((event) => event.deliveryStatus === 'PUBLISHED').length
  const publishedRate = recentAlerts.length === 0 ? 100 : Math.round(publishedAlerts / recentAlerts.length * 100)
  const alertRate = stats.processed ? Math.round(stats.alerts / stats.processed * 100) : 0

  const selectEvent = useCallback(async (event: EventRow) => {
    setSelectedID(event.fullId)
    try {
      const detail = await getTransaction(event.fullId)
      setAPIEvents((current) => current.map((item) => item.transaction_id === detail.transaction_id ? detail : item))
    } catch {
      // Polling keeps the list current if a detail request races with processing.
    }
  }, [])

  async function handleSubmit(transaction: SubmitTransaction) {
    const response = await submitTransaction(transaction)
    setPendingID(response.id)
    await loadDashboard()
  }

  return <div className="app-shell">
    <main>
      <DashboardHeader
        connected={connected}
        date={date}
        loadError={loadError}
        onGenerate={() => setSubmitOpen(true)}
        onSearchChange={setSearch}
        search={search}
        timezone={timezone}
      />
      <div className="fintech-grid" id="overview">
        <RiskActivityCard
          alertRate={alertRate}
          baselinePoints={`0,${baselineY} 630,${baselineY}`}
          chartPoints={chartPoints}
          connected={connected}
          latestY={latestY}
          onRangeChange={setRange}
          publishedRate={publishedRate}
          range={range}
          stats={stats}
        />
        <SignalDistributionCard stats={stats} />
        <EventHistoryCard
          connected={connected}
          events={filteredRows}
          filter={eventFilter}
          onFilterChange={setEventFilter}
          onSelect={(event) => void selectEvent(event)}
          pendingID={pendingID}
          selectedID={selected?.fullId}
        />
        <RiskCadenceCard events={rows} />
        <SelectedEventCard event={selected} />
      </div>
    </main>
    {submitOpen && <GenerateEventModal onClose={() => setSubmitOpen(false)} onSubmit={handleSubmit} />}
  </div>
}

export default App