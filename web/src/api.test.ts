import { describe, expect, it } from 'vitest'
import { chartPolyline, toEventRow, type TransactionEvent } from './api'

const event: TransactionEvent = {
  transaction_id: '5856ad7d-f529-4316-9b62-5e97e4e562cc',
  score: 100,
  severity: 'HIGH',
  triggered_rules: ['high_amount', 'sanctioned_jurisdiction'],
  processed_at: '2026-08-02T14:42:18Z',
  delivery_status: 'PUBLISHED',
  transaction: {
    id: '5856ad7d-f529-4316-9b62-5e97e4e562cc',
    user_id: '550e8400-e29b-41d4-a716-446655440000',
    amount: 99999.99,
    currency: 'EUR',
    ip: '203.0.113.15',
    country: 'KP',
    timestamp: '2026-08-02T14:42:17Z',
  },
}

describe('toEventRow', () => {
  it('maps an API event into the dashboard row', () => {
    const row = toEventRow(event)

    expect(row.id).toBe('5856ad7d')
    expect(row.fullId).toBe(event.transaction_id)
    expect(row.user).toBe('550e…0000')
    expect(row.amount).toBe('€99,999.99')
    expect(row.country).toBe('KP')
    expect(row.severity).toBe('HIGH')
    expect(row.deliveryStatus).toBe('PUBLISHED')
  })

  it('keeps legacy rows without a persisted snapshot renderable', () => {
    const row = toEventRow({ ...event, transaction: undefined, severity: undefined })

    expect(row.amount).toBe('—')
    expect(row.country).toBe('—')
    expect(row.user).toBe('snapshot unavailable')
    expect(row.severity).toBe('CLEAR')
  })
})

describe('chartPolyline', () => {
  it('builds bounded chart points in chronological order', () => {
    const points = chartPolyline([
      { ...event, score: 100 },
      { ...event, transaction_id: 'second', score: 0 },
    ])

    expect(points).toBe('0,150 630,35')
  })
})
