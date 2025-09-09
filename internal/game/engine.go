package game

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"gofigure/config"
	"gofigure/internal/logger"
	"gofigure/internal/ollama"
	"os"
	"strings"
	"time"
)

type Engine struct {
	murder       Murder
	ollamaClient *ollama.Client
	logger       *logger.Log
	config       *config.Config
}

func NewEngine(cfg *config.Config) (*Engine, error) {
	ollamaClient, err := ollama.NewClient(&cfg.Ollama)
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	return &Engine{
		ollamaClient: ollamaClient,
		logger:       logger.New(),
		config:       cfg,
	}, nil
}

func (e *Engine) WithMurder(filename string) *Engine {
	if filename == "" {
		e.logger.Warn("empty mystery filename")
		return e
	}

	m, err := loadMystery(filename)
	if err != nil {
		e.logger.WithError(err).Error("failed to load mystery")
		return e
	}
	e.murder = m
	return e
}

func (e *Engine) Start() error {
	// Check if Ollama model is available
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.ollamaClient.IsModelAvailable(ctx); err != nil {
		e.logger.WithError(err).Error("ollama model check failed")
		return fmt.Errorf("ollama setup error: %w", err)
	}

	e.logger.Info("ollama connection verified")
	fmt.Printf("🔍 Welcome Detective! You are investigating: %s\n", e.murder.Title)
	fmt.Println(e.murder.Intro)
	fmt.Println("\nType 'help' for available commands.")

	return e.gameLoop()
}

func (e *Engine) gameLoop() error {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, " ", 2)

		if len(parts) == 0 {
			continue
		}

		switch strings.ToLower(parts[0]) {
		case "help":
			e.showHelp()

		case "list":
			e.listCharacters()

		case "interview":
			if len(parts) < 2 {
				fmt.Println("Usage: interview <character>")
				continue
			}
			e.interviewCharacter(parts[1], scanner)

		case "accuse":
			if len(parts) < 2 {
				fmt.Println("Usage: accuse <name> <weapon> <location>")
				continue
			}
			if e.processAccusation(parts[1]) {
				return nil // Game won
			}

		case "quit", "exit":
			fmt.Println("Goodbye detective.")
			return nil

		default:
			fmt.Println("Unknown command. Type 'help' for options.")
		}
	}

	return nil
}

func (e *Engine) showHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("  help                           - Show this help message")
	fmt.Println("  list                           - List all characters")
	fmt.Println("  interview <character>          - Interview a character")
	fmt.Println("  accuse <name> <weapon> <location> - Make your final accusation")
	fmt.Println("  quit/exit                      - Exit the game")
}

func (e *Engine) listCharacters() {
	fmt.Println("\nCharacters in this mystery:")
	for _, char := range e.murder.Characters {
		fmt.Printf("  • %s (%s)\n", char.Name, char.Personality)
	}
	fmt.Println()
}

func (e *Engine) interviewCharacter(charName string, scanner *bufio.Scanner) {
	char := e.findCharacter(charName)
	if char == nil {
		fmt.Printf("No character named '%s' found. Type 'list' to see available characters.\n", charName)
		return
	}

	fmt.Printf("\n🎭 You are now interviewing %s\n", char.Name)
	fmt.Printf("Personality: %s\n", char.Personality)
	fmt.Println("Ask them questions (type 'exit' to stop):")

	for {
		fmt.Print("\nQ: ")
		if !scanner.Scan() {
			break
		}
		question := strings.TrimSpace(scanner.Text())
		if question == "exit" {
			fmt.Println("Interview ended.\n")
			break
		}
		if question == "" {
			continue
		}

		fmt.Print("🤔 Thinking...")

		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(e.config.Ollama.Timeout)*time.Second)

		answer, err := char.GenerateResponse(ctx, question, e.murder, e.ollamaClient)
		cancel()

		if err != nil {
			e.logger.WithError(err).Error("Failed to get character response")
			fmt.Printf("\n%s seems distracted and doesn't respond clearly.\n", char.Name)
			continue
		}

		fmt.Printf("\r%s: %s\n", char.Name, answer)
	}
}

func (e *Engine) findCharacter(name string) *Character {
	for i := range e.murder.Characters {
		if strings.Contains(strings.ToLower(e.murder.Characters[i].Name), strings.ToLower(name)) {
			return &e.murder.Characters[i]
		}
	}
	return nil
}

func (e *Engine) processAccusation(accusation string) bool {
	args := strings.Fields(accusation)
	if len(args) < 3 {
		fmt.Println("❌ Need to specify: name weapon location")
		return false
	}

	name, weapon, location := args[0], args[1], strings.Join(args[2:], " ")

	fmt.Printf("\n🔍 Your accusation: %s killed the victim with a %s in the %s\n",
		name, weapon, location)

	if strings.EqualFold(name, e.murder.Killer) &&
		strings.EqualFold(weapon, e.murder.Weapon) &&
		strings.EqualFold(location, e.murder.Location) {
		fmt.Println("🎉 Congratulations Detective! You solved the murder!")
		fmt.Printf("The killer was indeed %s with the %s in the %s.\n",
			e.murder.Killer, e.murder.Weapon, e.murder.Location)
		return true
	}

	fmt.Println("❌ Wrong accusation. The mystery continues...")
	fmt.Printf("💡 Hint: The actual solution was %s with the %s in the %s\n",
		e.murder.Killer, e.murder.Weapon, e.murder.Location)
	return false
}

func loadMystery(filename string) (Murder, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Murder{}, fmt.Errorf("failed to open mystery file: %w", err)
	}
	defer file.Close()

	var murder Murder
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&murder); err != nil {
		return Murder{}, fmt.Errorf("failed to decode mystery JSON: %w", err)
	}

	return murder, nil
}
