package config

import "strconv"

type HTTPConfig struct {
	Host string `env:"HOST" envDefault:"0.0.0.0"`
	Port int    `env:"PORT" envDefault:"8080"`
}

type PostgresConfig struct {
	// Either DSN directly (e.g. from AWS RDS secret),
	// or components to build it if DSN is empty.
	DSN      string `env:"DSN"`
	Host     string `env:"HOST"`
	Port     int    `env:"PORT" envDefault:"5432"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	DBName   string `env:"DBNAME"`
	SSLMode  string `env:"SSLMODE" envDefault:"disable"`
}

func (c PostgresConfig) EffectiveDSN() string {
	if c.DSN != "" {
		return c.DSN
	}
	// Very typical pgx DSN string; you can tweak as needed.
	return "postgres://" + c.User + ":" + c.Password +
		"@" + c.Host + ":" + strconv.Itoa(c.Port) +
		"/" + c.DBName + "?sslmode=" + c.SSLMode
}

type RedisConfig struct {
	Addr     string `env:"ADDR" envDefault:"localhost:6379"`
	Password string `env:"PASSWORD"`
	DB       int    `env:"DB" envDefault:"0"`
}

type KafkaConfig struct {
	Enabled     bool     `mapstructure:"enabled" yaml:"enabled"`
	Brokers     []string `mapstructure:"brokers" yaml:"brokers"`
	ClientID    string   `mapstructure:"client_id" yaml:"client_id"`
	GroupID     string   `mapstructure:"group_id" yaml:"group_id"`
	TopicPrefix string   `mapstructure:"topic_prefix" yaml:"topic_prefix"`
}

type SupplierConfig struct {
	Username string `env:"USERNAME,required"`
	Password string `env:"PASSWORD,required"`
}

// ObservabilityConfig Observability / telemetry configuration
type ObservabilityConfig struct {
	ServiceName string `env:"SERVICE_NAME" envDefault:"go-starter-api"`
	ServiceEnv  string `env:"SERVICE_ENV" envDefault:"Development"`
	// e.g. "http://otel-collector:4317"
	OtelEndpoint string `mapstructure:"otel_endpoint" yaml:"otel_endpoint"`
}

type Config struct {
	// Global environment, usually matches what you use in .NET: Development, Staging, Production...
	Environment string `env:"APP_ENV" envDefault:"Development"`

	HTTP          HTTPConfig          `envPrefix:"HTTP_"`
	Postgres      PostgresConfig      `envPrefix:"PG_"`
	Redis         RedisConfig         `envPrefix:"REDIS_"`
	Kafka         KafkaConfig         `envPrefix:"KAFKA_"`
	Supplier      SupplierConfig      `envPrefix:"SUPPLIER_"`
	Observability ObservabilityConfig `envPrefix:"OTEL_"`
}
