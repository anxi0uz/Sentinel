package configs

import (
	"reflect"
	"testing"

	"github.com/knadh/koanf/v2"
)

func TestEnvProviderParsesKafkaBrokers(t *testing.T) {
	t.Setenv("SENTINEL_KAFKA_BROKERS", "kafka-1:9092, kafka-2:9092")
	k := koanf.New(".")
	if err := k.Load(EnvProvider(), nil); err != nil {
		t.Fatal(err)
	}
	got, ok := k.Get("kafka.brokers").([]string)
	if !ok {
		t.Fatalf("kafka.brokers has type %T, want []string", k.Get("kafka.brokers"))
	}
	want := []string{"kafka-1:9092", "kafka-2:9092"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("brokers = %v, want %v", got, want)
	}
}
