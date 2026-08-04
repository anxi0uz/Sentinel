import { ArrowUpRight, X } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import type { SubmitTransaction } from '../api'

const demoUserID = '550e8400-e29b-41d4-a716-446655440000'
type Scenario = 'normal' | 'high_amount' | 'obvious_fraud'

type Props = {
  onClose: () => void
  onSubmit: (transaction: SubmitTransaction) => Promise<void>
}

export function GenerateEventModal({ onClose, onSubmit }: Props) {
  const [scenario, setScenario] = useState<Scenario>('obvious_fraud')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [form, setForm] = useState<SubmitTransaction>({ user_id: demoUserID, amount: 99999.99, currency: 'EUR', ip: '203.0.113.15', country: 'KP' })

  function applyScenario(next: Scenario) {
    const scenarios = {
      normal: { amount: 149, country: 'FI', ip: '198.51.100.24' },
      high_amount: { amount: 75000, country: 'FI', ip: '198.51.100.47' },
      obvious_fraud: { amount: 99999.99, country: 'KP', ip: '203.0.113.15' },
    }
    setScenario(next)
    setForm((current) => ({ ...current, ...scenarios[next] }))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError('')
    try {
      await onSubmit({ ...form, currency: form.currency.trim().toUpperCase(), country: form.country.trim().toUpperCase() })
      onClose()
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : 'Cannot submit transaction')
    } finally {
      setSubmitting(false)
    }
  }

  return <div className="modal-backdrop" role="presentation" onMouseDown={onClose}>
    <section className="event-modal" role="dialog" aria-modal="true" aria-labelledby="event-modal-title" onMouseDown={(event) => event.stopPropagation()}>
      <div className="modal-head">
        <div><span className="panel-kicker">Pipeline input</span><h2 id="event-modal-title">Generate transaction</h2></div>
        <button className="icon-button subtle" aria-label="Close" onClick={onClose}><X size={17} /></button>
      </div>
      <div className="scenario-row">
        <button type="button" className={scenario === 'normal' ? 'selected' : ''} aria-pressed={scenario === 'normal'} onClick={() => applyScenario('normal')}>Normal</button>
        <button type="button" className={scenario === 'high_amount' ? 'selected' : ''} aria-pressed={scenario === 'high_amount'} onClick={() => applyScenario('high_amount')}>High amount</button>
        <button type="button" className={scenario === 'obvious_fraud' ? 'selected' : ''} aria-pressed={scenario === 'obvious_fraud'} onClick={() => applyScenario('obvious_fraud')}>Obvious fraud</button>
      </div>
      <form onSubmit={(event) => void handleSubmit(event)}>
        <label className="wide">User ID<input required value={form.user_id} onChange={(event) => setForm((current) => ({ ...current, user_id: event.target.value }))} /></label>
        <label>Amount<input required min="0.01" step="0.01" type="number" value={form.amount} onChange={(event) => setForm((current) => ({ ...current, amount: Number(event.target.value) }))} /></label>
        <label>Currency<input required minLength={3} maxLength={3} value={form.currency} onChange={(event) => setForm((current) => ({ ...current, currency: event.target.value }))} /></label>
        <label>Country<input required minLength={2} maxLength={2} value={form.country} onChange={(event) => setForm((current) => ({ ...current, country: event.target.value }))} /></label>
        <label>IP address<input required value={form.ip} onChange={(event) => setForm((current) => ({ ...current, ip: event.target.value }))} /></label>
        {error && <p className="submit-error">{error}</p>}
        <div className="modal-actions"><button type="button" className="ghost-action" onClick={onClose}>Cancel</button><button className="generate-button" disabled={submitting}>{submitting ? 'Submitting…' : 'Drop into pipeline'} <ArrowUpRight size={14} /></button></div>
      </form>
    </section>
  </div>
}
