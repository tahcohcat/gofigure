package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gofigure/internal/domain/game"
	"gofigure/internal/logger"
	"os"
	"strings"
)

type Engine struct {
	murder game.Murder
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) WithMurder(filename string) *Engine {
	if filename != "" {
		logger.New().Warn("empty mystery filename")
	}
	m, err := loadMystery(filename)
	if err != nil {
		logger.New().WithError(err).Error("failed to load mystery")
		return e
	}
	e.murder = m
	return e
}

func (e *Engine) Start() {

	fmt.Printf("🔍 Welcome Detective! You are investigating: %s\n", e.murder.Title)
	fmt.Println("Type 'help' for available commands.")

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
			fmt.Println("Commands:")
			fmt.Println("  interview <character>  - Talk to a character")
			fmt.Println("  accuse <name> <weapon> <location> - Make an accusation")
			fmt.Println("  quit                   - Exit the game")

		case "interview":
			if len(parts) < 2 {
				fmt.Println("Usage: interview <character>")
				continue
			}
			charName := strings.ToLower(parts[1])
			var char *game.Character
			for i := range e.murder.Characters {
				if strings.ToLower(e.murder.Characters[i].Name) == charName {
					char = &e.murder.Characters[i]
					break
				}
			}
			if char == nil {
				fmt.Println("No such character.")
				continue
			}

			fmt.Printf("You are now interviewing %s. Ask them questions (type 'exit' to stop):\n", char.Name)
			for {
				fmt.Print("Q: ")
				if !scanner.Scan() {
					break
				}
				q := strings.TrimSpace(scanner.Text())
				if q == "exit" {
					break
				}
				answer := char.GeneratePrompt(q, e.murder)
				fmt.Printf("%s: %s\n", char.Name, answer)
			}

		case "accuse":
			if len(parts) < 2 {
				fmt.Println("Usage: accuse <name> <weapon> <location>")
				continue
			}
			args := strings.Split(parts[1], " ")
			if len(args) < 3 {
				fmt.Println("Need name, weapon, and location.")
				continue
			}
			name, weapon, location := args[0], args[1], args[2]
			if strings.EqualFold(name, e.murder.Killer) &&
				strings.EqualFold(weapon, e.murder.Weapon) &&
				strings.EqualFold(location, e.murder.Location) {
				fmt.Println("🎉 You solved the murder! Well done detective.")
			} else {
				fmt.Println("❌ Wrong accusation. The murderer remains at large...")
			}
			return

		case "quit":
			fmt.Println("Goodbye detective.")
			return

		default:
			fmt.Println("Unknown command. Type 'help' for options.")
		}
	}
}

// Load a murder mystery from JSON file
func loadMystery(filename string) (game.Murder, error) {
	file, err := os.Open(filename)
	if err != nil {
		return game.Murder{}, err
	}
	defer file.Close()

	var murder game.Murder
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&murder); err != nil {
		return game.Murder{}, err
	}

	return murder, nil
}
