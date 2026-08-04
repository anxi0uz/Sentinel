import { describe, expect, it } from 'vitest'
import { matchedRuleValue, ruleLabel } from './rules'

describe('rule presentation', () => {
  it('labels the aggregate by rule name', () => {
    expect(ruleLabel('sanctioned_jurisdiction')).toBe('Sanctioned jurisdiction')
    expect(ruleLabel('north_korea')).toBe('Sanctioned jurisdiction')
  })

  it('shows the sanctioned country only as a transaction match', () => {
    expect(matchedRuleValue('sanctioned_jurisdiction', 'KP')).toBe('North Korea (KP)')
    expect(matchedRuleValue('high_amount', 'KP')).toBeUndefined()
  })
})
