package game

import (
	"fmt"
	"os/exec"
)

// Character in the game
type Character struct {
	Name        string   `json:"name"`
	Personality string   `json:"personality"`
	Knowledge   []string `json:"knowledge"`
	Reliable    bool     `json:"reliable"`
}

// Ask Ollama for a character response
func (c Character) GeneratePrompt(question string, murder Murder) string {
	prompt := fmt.Sprintf(`You are roleplaying as %s.
Personality: %s
You may know some things about a murder.
Murder details: killer=%s, weapon=%s, location=%s
Your knowledge: %v
If you are unreliable, sometimes hide facts or contradict slightly.
Answer the player's question in character.

Player asked: %s`,
		c.Name, c.Personality, murder.Killer, murder.Weapon, murder.Location, c.Knowledge, question)

	cmd := exec.Command("ollama", "run", "llama3", prompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error talking to %s: %v", c.Name, err)
	}
	return string(out)
}
