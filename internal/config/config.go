package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config defines the application configuration structure
type Config struct {
	Auth       AuthConfig       `mapstructure:"auth"`
	Broker     BrokerConfig     `mapstructure:"broker"`
	Historical HistoricalConfig `mapstructure:"historical"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	AuthServiceURL    string `mapstructure:"auth_service_url"`
	AuthServiceAPIKey string `mapstructure:"auth_service_api_key"`
	BrokerName        string `mapstructure:"broker_name"`
	ApiKey            string `mapstructure:"api_key"`
	ApiSecret         string `mapstructure:"api_secret"`
	SessionToken      string `mapstructure:"session_token"`
}

// BrokerConfig defines the broker configuration
type BrokerConfig struct {
	InstrumentsNSEURL string `mapstructure:"instruments_nse_url"`
}

// HistoricalConfig defines the historical data download configuration
type HistoricalConfig struct {
	OutputDir       string `mapstructure:"output_dir"`
	ParquetEnabled  bool   `mapstructure:"parquet_enabled"`
	ParquetDir      string `mapstructure:"parquet_dir"`
	Interval        string `mapstructure:"interval"`
	DaysToFetch     int    `mapstructure:"days_to_fetch"`
	RequestDelay    int    `mapstructure:"request_delay"`
	MaxRetries      int    `mapstructure:"max_retries"`
	InstrumentsPath string `mapstructure:"instruments_path"`
}

// LoadConfig loads configuration from file and overrides with environment variables
func LoadConfig(path string) (Config, error) {
	// Set up Viper to first try to read from config file
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	// Set up environment variable prefixes
	viper.SetEnvPrefix("HISTORICAL")

	// Set up mappings for nested config keys to env vars
	// Auth mappings
	viper.BindEnv("auth.auth_service_url", "HISTORICAL_AUTH_SERVICE_URL")
	viper.BindEnv("auth.auth_service_api_key", "HISTORICAL_AUTH_SERVICE_KEY")
	viper.BindEnv("auth.broker_name", "HISTORICAL_BROKER_NAME")
	viper.BindEnv("auth.api_key", "HISTORICAL_API_KEY")
	viper.BindEnv("auth.api_secret", "HISTORICAL_API_SECRET")
	viper.BindEnv("auth.session_token", "HISTORICAL_SESSION_TOKEN")

	// Broker mappings
	viper.BindEnv("broker.instruments_nse_url", "HISTORICAL_INSTRUMENTS_NSE_URL")

	// Historical data mappings
	viper.BindEnv("historical.output_dir", "HISTORICAL_OUTPUT_DIR")
	viper.BindEnv("historical.parquet_enabled", "HISTORICAL_PARQUET_ENABLED")
	viper.BindEnv("historical.parquet_dir", "HISTORICAL_PARQUET_DIR")
	viper.BindEnv("historical.interval", "HISTORICAL_INTERVAL")
	viper.BindEnv("historical.days_to_fetch", "HISTORICAL_DAYS")
	viper.BindEnv("historical.request_delay", "HISTORICAL_REQUEST_DELAY")
	viper.BindEnv("historical.max_retries", "HISTORICAL_MAX_RETRIES")
	viper.BindEnv("historical.instruments_path", "HISTORICAL_INSTRUMENTS_PATH")

	// First attempt to read the config file
	var configFileFound bool
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Printf("Config file not found at %s, falling back to environment variables\n", path)
		} else {
			fmt.Printf("Error reading config file %s: %v, falling back to environment variables\n", path, err)
		}
	} else {
		configFileFound = true
		fmt.Printf("Loaded config from %s, will override with environment variables\n", viper.ConfigFileUsed())
	}

	// IMPORTANT: Enable automatic environment variable binding AFTER reading config file
	// This ensures environment variables take precedence over config file values
	viper.AutomaticEnv()

	// Create default config structure
	var config Config

	// Unmarshal the config
	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Apply default values for any settings not specified
	applyDefaults(&config)

	// Log loading status
	if configFileFound {
		fmt.Println("Configuration loaded from file and overridden with environment variables")
	} else {
		fmt.Println("Configuration loaded from environment variables with defaults applied")
	}

	return config, nil
}

// applyDefaults sets default values for any config values not set from file or environment
func applyDefaults(config *Config) {
	// Auth defaults
	if config.Auth.BrokerName == "" {
		config.Auth.BrokerName = "zerodha"
	}

	// Broker defaults
	if config.Broker.InstrumentsNSEURL == "" {
		config.Broker.InstrumentsNSEURL = "https://api.kite.trade/instruments/NSE"
	}

	// Historical data defaults
	if config.Historical.OutputDir == "" {
		config.Historical.OutputDir = "./historical_data"
	}
	if config.Historical.ParquetDir == "" {
		config.Historical.ParquetDir = "./parquet_data"
	}
	if config.Historical.Interval == "" {
		config.Historical.Interval = "minute"
	}
	if config.Historical.DaysToFetch == 0 {
		config.Historical.DaysToFetch = 30
	}
	if config.Historical.RequestDelay == 0 {
		config.Historical.RequestDelay = 500
	}
	if config.Historical.MaxRetries == 0 {
		config.Historical.MaxRetries = 3
	}
	if config.Historical.InstrumentsPath == "" {
		config.Historical.InstrumentsPath = "./instruments.csv"
	}
}