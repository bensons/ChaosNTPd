package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the ChaosNTPd configuration
type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
		Name string `yaml:"name"`
	} `yaml:"server"`

	NTP struct {
		Stratum     int    `yaml:"stratum"`
		ReferenceID string `yaml:"reference_id"`
		Precision   int    `yaml:"precision"`
	} `yaml:"ntp"`

	TimeManipulation struct {
		InitialOffsetMinutes int    `yaml:"initial_offset_minutes"`
		JitterSeconds        int    `yaml:"jitter_seconds"`
		Distribution         string `yaml:"distribution"`

		ClientTracking struct {
			CleanupIntervalSeconds int `yaml:"cleanup_interval_seconds"`
			MaxClientAgeSeconds    int `yaml:"max_client_age_seconds"`
			MaxTrackedClients      int `yaml:"max_tracked_clients"`
		} `yaml:"client_tracking"`
	} `yaml:"time_manipulation"`

	Logging struct {
		Level           string   `yaml:"level"`
		Format          string   `yaml:"format"`
		LogTransactions bool     `yaml:"log_transactions"`
		Output          string   `yaml:"output"`
	} `yaml:"logging"`

	Security struct {
		AllowList []string `yaml:"allow_list"`
		RateLimit struct {
			Enabled              bool `yaml:"enabled"`
			MaxRequestsPerMinute int  `yaml:"max_requests_per_minute"`
		} `yaml:"rate_limit"`
	} `yaml:"security"`
}

// CLIFlags holds command-line flag values
type CLIFlags struct {
	ConfigPath    string
	InitialOffset int
	Jitter        int
	Stratum       int
	Port          int
	Host          string
	LogLevel      string
}

// ParseFlags parses command-line arguments
func ParseFlags() *CLIFlags {
	flags := &CLIFlags{}

	flag.StringVar(&flags.ConfigPath, "config", "config.yaml", "Path to configuration file")
	flag.StringVar(&flags.ConfigPath, "c", "config.yaml", "Path to configuration file (shorthand)")

	flag.IntVar(&flags.InitialOffset, "initial-offset", -1, "Initial offset in minutes (overrides config)")
	flag.IntVar(&flags.InitialOffset, "N", -1, "Initial offset in minutes (shorthand)")

	flag.IntVar(&flags.Jitter, "jitter", -1, "Jitter in seconds (overrides config)")
	flag.IntVar(&flags.Jitter, "X", -1, "Jitter in seconds (shorthand)")

	flag.IntVar(&flags.Stratum, "stratum", -1, "NTP stratum level 0-15 (overrides config)")
	flag.IntVar(&flags.Stratum, "s", -1, "NTP stratum level (shorthand)")

	flag.IntVar(&flags.Port, "port", -1, "UDP port (overrides config)")
	flag.IntVar(&flags.Port, "p", -1, "UDP port (shorthand)")

	flag.StringVar(&flags.Host, "host", "", "Bind address (overrides config)")
	flag.StringVar(&flags.LogLevel, "log-level", "", "Logging level (overrides config)")

	flag.Parse()

	return flags
}

// LoadConfig loads configuration from file and applies CLI overrides
func LoadConfig(configPath string, flags *CLIFlags) (*Config, error) {
	// Set defaults
	config := &Config{}
	config.Server.Host = "0.0.0.0"
	config.Server.Port = 123
	config.Server.Name = "ChaosNTPd"

	config.NTP.Stratum = 1
	config.NTP.ReferenceID = "CHAO"
	config.NTP.Precision = -20

	config.TimeManipulation.InitialOffsetMinutes = 30
	config.TimeManipulation.JitterSeconds = 5
	config.TimeManipulation.Distribution = "uniform"
	config.TimeManipulation.ClientTracking.CleanupIntervalSeconds = 300
	config.TimeManipulation.ClientTracking.MaxClientAgeSeconds = 3600
	config.TimeManipulation.ClientTracking.MaxTrackedClients = 10000

	config.Logging.Level = "INFO"
	config.Logging.Format = "json"
	config.Logging.LogTransactions = true
	config.Logging.Output = "stdout"

	// Load config file if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("error parsing config file: %w", err)
		}
	}

	// Apply CLI overrides
	if flags.InitialOffset >= 0 {
		config.TimeManipulation.InitialOffsetMinutes = flags.InitialOffset
	}
	if flags.Jitter >= 0 {
		config.TimeManipulation.JitterSeconds = flags.Jitter
	}
	if flags.Stratum >= 0 {
		config.NTP.Stratum = flags.Stratum
	}
	if flags.Port > 0 {
		config.Server.Port = flags.Port
	}
	if flags.Host != "" {
		config.Server.Host = flags.Host
	}
	if flags.LogLevel != "" {
		config.Logging.Level = flags.LogLevel
	}

	// Validate
	if config.NTP.Stratum < 0 || config.NTP.Stratum > 15 {
		return nil, fmt.Errorf("invalid stratum value: %d (must be 0-15)", config.NTP.Stratum)
	}

	return config, nil
}

// PrintStartupBanner prints the startup information
func PrintStartupBanner(config *Config) {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        ChaosNTPd v1.0                          ║")
	fmt.Println("║           Adversarial NTP Daemon for Testing                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("⚠️  WARNING: This server distributes INACCURATE time information!")
	fmt.Println("⚠️  Deploy ONLY in isolated test environments!")
	fmt.Println()
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Listening:      %s:%d\n", config.Server.Host, config.Server.Port)
	fmt.Printf("  Stratum:        %d (0=invalid, 1=primary, 2-15=secondary)\n", config.NTP.Stratum)
	fmt.Printf("  Reference ID:   %s\n", config.NTP.ReferenceID)
	fmt.Printf("  Initial Offset: ±%d minutes\n", config.TimeManipulation.InitialOffsetMinutes)
	fmt.Printf("  Jitter:         ±%d seconds\n", config.TimeManipulation.JitterSeconds)
	fmt.Printf("  Distribution:   %s\n", config.TimeManipulation.Distribution)
	fmt.Printf("  Log Format:     %s\n", config.Logging.Format)
	fmt.Println()
	fmt.Println("Starting server...")
	fmt.Println()
}
