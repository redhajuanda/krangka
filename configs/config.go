package configs

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents configuration variables
type Config struct {
	App      App      `yaml:"app"`
	Log      Log      `yaml:"log"`
	Http     Http     `yaml:"http"`
	Database Database `yaml:"database"`
	Cache    Cache    `yaml:"cache"`
	Otel     Otel     `yaml:"otel"`
	Roles    Roles    `yaml:"roles"`
	Event    Event    `yaml:"event"`
}

type App struct {
	Name string `yaml:"name"`
}

type Log struct {
	Level          uint32   `yaml:"level"`
	Format         string   `yaml:"format"`
	RedactedFields []string `yaml:"redacted_fields"`
}

type Http struct {
	Port                  string        `yaml:"port"`
	ReadTimeout           time.Duration `yaml:"read_timeout"`
	WriteTimeout          time.Duration `yaml:"write_timeout"`
	IdleTimeout           time.Duration `yaml:"idle_timeout"`
	StartTimeout          time.Duration `yaml:"start_timeout"`
	StopTimeout           time.Duration `yaml:"stop_timeout"`
	EnablePrintRoutes     bool          `yaml:"enable_print_routes"`
	DisableStartupMessage bool          `yaml:"disable_startup_message"`
}

type Database struct {
	MariaDBMain struct {
		Host            string        `yaml:"host"`
		Port            string        `yaml:"port"`
		Username        string        `yaml:"username"`
		Password        string        `yaml:"password"`
		DBName          string        `yaml:"name"`
		MaxOpenConns    int           `yaml:"max_open_conns"`
		MaxIdleConns    int           `yaml:"max_idle_conns"`
		ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
		ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	} `yaml:"mariadb_main"`
	MariaDBWorker struct {
		Host            string        `yaml:"host"`
		Port            string        `yaml:"port"`
		Username        string        `yaml:"username"`
		Password        string        `yaml:"password"`
		DBName          string        `yaml:"name"`
		MaxOpenConns    int           `yaml:"max_open_conns"`
		MaxIdleConns    int           `yaml:"max_idle_conns"`
		ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
		ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	} `yaml:"mariadb_worker"`
}

type Cache struct {
	Redis struct {
		Sentinel     bool     `yaml:"sentinel"`
		MasterName   string   `yaml:"master_name"`
		Username     string   `yaml:"username"`
		Hosts        []string `yaml:"hosts"`
		Password     string   `yaml:"password"`
		DB           int      `yaml:"db"`
		PoolSize     int      `yaml:"pool_size"`
		MinIdleConns int      `yaml:"min_idle_conns"`
	} `yaml:"redis"`
}
type Roles struct {
	Superadmin string `yaml:"superadmin"`
}

type Otel struct {
	URL        string  `yaml:"url"`
	Exporter   string  `yaml:"exporter"`
	SampleRate float64 `yaml:"sample_rate"`
}

type Event struct {
	Outbox struct {
		RelayPattern     string `yaml:"relay_pattern"`
		MaxRetryAttempts int    `yaml:"max_retry_attempts"`
		FetchPerPage     int    `yaml:"fetch_per_page"`
	} `yaml:"outbox"`
	Idempotency struct {
		TTL time.Duration `yaml:"ttl"`
	} `yaml:"idempotency"`
	Redisstream struct {
		Publisher   RedisstreamPublisher   `yaml:"publisher"`
		Subscribers RedisstreamSubscribers `yaml:"subscribers"`
	} `yaml:"redisstream"`
	Kafka struct {
		Publisher   KafkaPublisher   `yaml:"publisher"`
		Subscribers KafkaSubscribers `yaml:"subscribers"`
	} `yaml:"kafka"`
}

type RedisstreamPublisher struct {
	DefaultMaxlen int64 `yaml:"default_maxlen"`
}

type RedisstreamSubscriber struct {
	ID            string `yaml:"id"`
	ConsumerGroup string `yaml:"consumer_group"`
}

type RedisstreamSubscribers []RedisstreamSubscriber

// GetByID gets a subscriber by its ID
func (r RedisstreamSubscribers) GetByID(id string) *RedisstreamSubscriber {
	for _, subscriber := range r {
		if subscriber.ID == id {
			return &subscriber
		}
	}
	return nil
}

type KafkaPublisher struct {
	Brokers      []string `yaml:"brokers"`
	DebugEnabled bool     `yaml:"debug_enabled"`
	TraceEnabled bool     `yaml:"trace_enabled"`
}
type KafkaSubscriber struct {
	ID            string   `yaml:"id"`
	Brokers       []string `yaml:"brokers"`
	ConsumerGroup string   `yaml:"consumer_group"`
	DebugEnabled  bool     `yaml:"debug_enabled"`
	TraceEnabled  bool     `yaml:"trace_enabled"`
}

type KafkaSubscribers []KafkaSubscriber

// GetByID gets a subscriber by its ID
func (k KafkaSubscribers) GetByID(id string) *KafkaSubscriber {
	for _, subscriber := range k {
		if subscriber.ID == id {
			return &subscriber
		}
	}
	return nil
}

// GetEnv returns the environment variable SCENV
func (c *Config) GetEnv() Env {
	return Env(os.Getenv("SCENV"))
}

// LoadConfig loads the configuration from the given file path
func LoadConfig(cfgFile string) *Config {

	var cfg Config

	// read file cfgFile
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		log.Fatalf("read config error: %v", err)
	}

	// unmarshal yaml to config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("unmarshal yaml error: %v", err)
	}

	return &cfg
}
