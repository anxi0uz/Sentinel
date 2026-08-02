package scorer

import (
	"slices"

	"github.com/anxi0uz/sentinel/pkg/models"
)

func Score(rules []models.FraudRule, tx models.EnrichedTransaction) (int, []string) {
	var total int
	var triggered []string

	for _, rule := range rules {
		if !rule.Active {
			continue
		}
		if applies(rule, tx) {
			total += int(rule.ScoreDelta)
			triggered = append(triggered, rule.Name)
		}
	}
	return total, triggered
}

func applies(rule models.FraudRule, tx models.EnrichedTransaction) bool {
	switch rule.Operator {
	case "gt":
		value, ok := numericField(rule.Field, tx)
		return ok && value > rule.Threshold
	case "lt":
		value, ok := numericField(rule.Field, tx)
		return ok && value < rule.Threshold
	case "eq":
		if len(rule.Values) == 0 {
			return false
		}
		value, ok := stringField(rule.Field, tx)
		return ok && value == rule.Values[0]
	case "not_in":
		value, ok := stringField(rule.Field, tx)
		return ok && !slices.Contains(rule.Values, value)
	case "impossible_travel":
		return impossibleTravel(tx, rule.Threshold)
	}
	return false
}

func numericField(field string, tx models.EnrichedTransaction) (float64, bool) {
	switch field {
	case "amount":
		return tx.Transaction.Amount, true
	}
	return 0, false
}

func stringField(field string, tx models.EnrichedTransaction) (string, bool) {
	switch field {
	case "country":
		return tx.Transaction.Country, true
	case "ip":
		return tx.Transaction.IP, true
	}
	return "", false
}

func impossibleTravel(tx models.EnrichedTransaction, minHours float64) bool {
	if minHours <= 0 || tx.User.LastSeenAt.IsZero() || tx.Transaction.Timestamp.IsZero() {
		return false
	}
	if tx.Transaction.Country == "" || tx.User.LastCountry == "" || tx.Transaction.Country == tx.User.LastCountry {
		return false
	}
	if !tx.Transaction.Timestamp.After(tx.User.LastSeenAt) {
		return false
	}
	elapsed := tx.Transaction.Timestamp.Sub(tx.User.LastSeenAt).Hours()
	return elapsed < minHours
}
