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
		return numericField(rule.Field, tx) > rule.Threshold
	case "lt":
		return numericField(rule.Field, tx) < rule.Threshold
	case "eq":
		if len(rule.Values) == 0 {
			return false
		}
		return stringField(rule.Field, tx) == rule.Values[0]
	case "not_in":
		v := stringField(rule.Field, tx)
		return !slices.Contains(rule.Values, v)
	case "impossible_travel":
		return impossibleTravel(tx, rule.Threshold)
	}
	return false
}

func numericField(field string, tx models.EnrichedTransaction) float64 {
	switch field {
	case "amount":
		return tx.Transaction.Amount
	}
	return 0
}

func stringField(field string, tx models.EnrichedTransaction) string {
	switch field {
	case "country":
		return tx.Transaction.Country
	case "ip":
		return tx.Transaction.IP
	}
	return ""
}

func impossibleTravel(tx models.EnrichedTransaction, minHours float64) bool {
	if tx.Transaction.Country == tx.User.LastCountry {
		return false
	}
	elapsed := tx.Transaction.Timestamp.Sub(tx.User.LastSeenAt).Hours()
	return elapsed < minHours
}
