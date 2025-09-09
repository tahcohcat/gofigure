package main

import (
	"fmt"
	"gofigure/config"
	"gofigure/internal/game"
	"gofigure/internal/logger"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile  string
	showResp bool
	cfg      *config.Config
	log      = logger.New()
)

var rootCmd = &cobra.Command{
	Use:   "gofigure",
	Short: "Murder Mystery CLI game powered by Ollama",
	Long:  "A CLI murder mystery roleplay engine powered by Ollama AI and JSON scenario definitions.",
}

var playCmd = &cobra.Command{
	Use:   "play [mystery.json]",
	Short: "Play a murder mystery",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mysteryFile := args[0]

		e, err := game.NewEngine(cfg)
		if err != nil {
			return fmt.Errorf("failed to create engine: %w", err)
		}

		return e.WithMurder(mysteryFile).WithResponses(showResp).Start()
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Current Configuration:\n")
		fmt.Printf("  Ollama Host: %s\n", cfg.Ollama.Host)
		fmt.Printf("  Ollama Model: %s\n", cfg.Ollama.Model)
		fmt.Printf("  Timeout: %d seconds\n", cfg.Ollama.Timeout)
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&showResp, "show-responses", false, "show responses in output")
}

func initConfig() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		log.WithError(err).Error("Failed to load configuration")
		os.Exit(1)
	}
}

func main() {
	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(configCmd)

	if err := rootCmd.Execute(); err != nil {
		log.WithError(err).Error("Command execution failed")
		os.Exit(1)
	}
}
