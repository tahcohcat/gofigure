package main

import (
	"fmt"
	"gofigure/internal/engine"
	"os"

	"github.com/spf13/cobra"
)

var (
	startGame = func(cmd *cobra.Command, args []string) {
		mysteryFile := args[0]
		e := engine.NewEngine().WithMurder(mysteryFile)
		e.Start()
	}
)

var rootCmd = &cobra.Command{
	Use:   "mystery",
	Short: "Murder Mystery CLI game",
	Long:  "A CLI murder mystery roleplay engine powered by Ollama and a JSON scenario definition.",
}

var playCmd = &cobra.Command{
	Use:   "play [mystery.json]",
	Short: "Play a murder mystery",
	Args:  cobra.ExactArgs(1),
	Run:   startGame,
}

var listCmd = &cobra.Command{
	Use:   "list [mystery.json]",
	Short: "List characters in a mystery",
	Args:  cobra.ExactArgs(1),
	Run:   startGame,
}

func main() {
	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(listCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
