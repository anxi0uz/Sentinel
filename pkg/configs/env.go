package configs

import (
	"strings"

	"github.com/knadh/koanf/providers/env"
)

func EnvProvider() *env.Env {
	return env.ProviderWithValue("SENTINEL_", ".", func(name, value string) (string, interface{}) {
		key := strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(name, "SENTINEL_")), "_", ".")
		if key != "kafka.brokers" {
			return key, value
		}

		parts := strings.Split(value, ",")
		brokers := make([]string, 0, len(parts))
		for _, part := range parts {
			if broker := strings.TrimSpace(part); broker != "" {
				brokers = append(brokers, broker)
			}
		}
		return key, brokers
	})
}
