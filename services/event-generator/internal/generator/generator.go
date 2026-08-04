package generator

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/google/uuid"
)

type Scenario string

const (
	ScenarioMixed          Scenario = "mixed"
	ScenarioNormal         Scenario = "normal"
	ScenarioHighAmount     Scenario = "high_amount"
	ScenarioBlockedCountry Scenario = "blocked_country"
	ScenarioObviousFraud   Scenario = "obvious_fraud"
)

func ParseScenario(value string) Scenario {
	switch Scenario(strings.ToLower(strings.TrimSpace(value))) {
	case ScenarioNormal:
		return ScenarioNormal
	case ScenarioHighAmount:
		return ScenarioHighAmount
	case ScenarioBlockedCountry:
		return ScenarioBlockedCountry
	case ScenarioObviousFraud:
		return ScenarioObviousFraud
	default:
		return ScenarioMixed
	}
}

func NewEvent(scenario Scenario, now time.Time) models.EnrichedTransaction {
	if scenario == ScenarioMixed {
		roll := rand.IntN(100)
		switch {
		case roll < 75:
			scenario = ScenarioNormal
		case roll < 85:
			scenario = ScenarioHighAmount
		case roll < 95:
			scenario = ScenarioBlockedCountry
		default:
			scenario = ScenarioObviousFraud
		}
	}

	userID := uuid.New()
	country := []string{"FI", "SE", "DE", "NL", "FR"}[rand.IntN(5)]
	amount := float64(rand.IntN(199901)+100) / 100

	switch scenario {
	case ScenarioHighAmount:
		amount = float64(rand.IntN(2500000)+5000001) / 100
	case ScenarioBlockedCountry:
		country = "KP"
	case ScenarioObviousFraud:
		amount = float64(rand.IntN(2500000)+5000001) / 100
		country = "KP"
	}

	return models.EnrichedTransaction{
		Transaction: models.Transaction{
			ID:        uuid.New(),
			UserID:    userID,
			Amount:    amount,
			Currency:  "EUR",
			IP:        randomIP(),
			Country:   country,
			Timestamp: now.UTC(),
		},
		User: models.User{
			ID:          userID,
			Country:     country,
			LastIP:      randomIP(),
			LastCountry: country,
			LastSeenAt:  now.Add(-time.Duration(rand.IntN(48)+1) * time.Hour).UTC(),
			CreatedAt:   now.Add(-time.Duration(rand.IntN(720)+24) * time.Hour).UTC(),
		},
	}
}

func randomIP() string {
	return "203.0.113." + strconv.Itoa(rand.IntN(253)+1)
}
