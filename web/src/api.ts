export type Severity = 'CLEAR' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
export type DeliveryStatus = 'NOT_REQUIRED' | 'PENDING' | 'PUBLISHED'

export type TransactionSnapshot = {
  id: string
  user_id: string
  amount: number
  currency: string
  ip: string
  country: string
  timestamp: string
}

export type UserSnapshot = {
  id: string
  country: string
  last_ip: string
  last_country: string
  last_seen_at: string
  created_at: string
}

export type TransactionEvent = {
  alert_id?: string
  delivery_status: DeliveryStatus
  processed_at: string
  score: number
  severity?: Exclude<Severity, 'CLEAR'>
  transaction?: TransactionSnapshot
  transaction_id: string
  triggered_rules: string[]
  user?: UserSnapshot
}

export type StatsResponse = {
  alerts: number
  average_score: number
  by_severity: {
    medium: number
    high: number
    critical: number
  }
  processed: number
  top_rules: Array<{ name: string; count: number }>
}

export type TransactionListResponse = {
  items: TransactionEvent[]
  pagination: {
    limit: number
    offset: number
    total: number
  }
}

export type SubmitTransaction = {
  user_id: string
  amount: number
  currency: string
  ip: string
  country: string
}

export type EventRow = {
  id: string
  fullId: string
  user: string
  amount: string
  country: string
  score: number
  severity: Severity
  rules: string[]
  time: string
  deliveryStatus: DeliveryStatus
  source: TransactionEvent
}

export class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

const apiBase = '/api'

export async function getTransactions(signal?: AbortSignal): Promise<TransactionListResponse> {
  return request<TransactionListResponse>('/transactions?limit=100', { signal })
}

export async function getTransaction(id: string, signal?: AbortSignal): Promise<TransactionEvent> {
  return request<TransactionEvent>(`/transactions/${encodeURIComponent(id)}`, { signal })
}

export async function getStats(signal?: AbortSignal): Promise<StatsResponse> {
  return request<StatsResponse>('/stats', { signal })
}

export async function submitTransaction(transaction: SubmitTransaction): Promise<{ id: string }> {
  return request<{ id: string }>('/transactions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(transaction),
  })
}

export function toEventRow(event: TransactionEvent): EventRow {
  const transaction = event.transaction
  return {
    id: event.transaction_id.slice(0, 8),
    fullId: event.transaction_id,
    user: transaction ? shortID(transaction.user_id) : 'snapshot unavailable',
    amount: transaction ? formatAmount(transaction.amount, transaction.currency) : '—',
    country: transaction?.country || '—',
    score: event.score,
    severity: event.severity ?? 'CLEAR',
    rules: event.triggered_rules ?? [],
    time: new Intl.DateTimeFormat('en-GB', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(new Date(event.processed_at)),
    deliveryStatus: event.delivery_status,
    source: event,
  }
}

export function chartPolyline(events: TransactionEvent[]): string {
  const chronological = events.slice(0, 16).reverse()
  if (chronological.length === 0) {
    return '0,150 630,150'
  }
  const step = chronological.length === 1 ? 0 : 630 / (chronological.length - 1)
  return chronological
    .map((event, index) => {
      const score = Math.max(0, Math.min(120, event.score))
      const y = Math.round(150 - score * 1.15)
      return `${Math.round(index * step)},${y}`
    })
    .join(' ')
}

function shortID(id: string): string {
  if (id.length < 9) {
    return id
  }
  return `${id.slice(0, 4)}…${id.slice(-4)}`
}

function formatAmount(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat('en-IE', {
      style: 'currency',
      currency,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(amount)
  } catch {
    return `${amount.toFixed(2)} ${currency}`
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, init)
  if (!response.ok) {
    let message = `Request failed with status ${response.status}`
    try {
      const body = await response.json() as { error?: string }
      if (body.error) {
        message = body.error
      }
    } catch {
      // Keep the status-based message for non-JSON failures.
    }
    throw new APIError(response.status, message)
  }
  return response.json() as Promise<T>
}
