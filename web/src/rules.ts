const ruleLabels: Record<string, string> = {
  high_amount: 'High amount',
  north_korea: 'Sanctioned jurisdiction',
  sanctioned_jurisdiction: 'Sanctioned jurisdiction',
}

export function ruleLabel(rule: string): string {
  return ruleLabels[rule] ?? rule
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

export function matchedRuleValue(rule: string, country: string): string | undefined {
  if (rule !== 'sanctioned_jurisdiction' && rule !== 'north_korea') return
  return country === 'KP' ? 'North Korea (KP)' : country
}
