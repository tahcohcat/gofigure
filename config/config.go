package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Ollama OllamaConfig `mapstructure:"ollama"`
	Tts    TtsConfig    `mapstructure:"tts"`
	Sst    SstConfig    `mapstructure:"sst"`
}

type TtsConfig struct {
	Type    string `mapstructure:"type"`
	Enabled bool   `mapstructure:"enabled"`
}

type SstConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Provider     string `mapstructure:"provider"`
	LanguageCode string `mapstructure:"language_code"`
	SampleRate   int    `mapstructure:"sample_rate"`
}

type OllamaConfig struct {
	Host    string `mapstructure:"host"`
	Model   string `mapstructure:"model"`
	Timeout int    `mapstructure:"timeout"` // seconds
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("ollama.host", "http://localhost:11434")
	viper.SetDefault("ollama.model", "llama3.2")
	viper.SetDefault("ollama.timeout", 50)

	viper.SetDefault("tts.enabled", true)
	viper.SetDefault("tts.type", "google")

	viper.SetDefault("sst.enabled", true)
	viper.SetDefault("sst.provider", "google")
	viper.SetDefault("sst.language_code", "en-US")
	viper.SetDefault("sst.sample_rate", 16000)

	// Allow environment variables
	viper.SetEnvPrefix("GOFIGURE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found, use defaults
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
