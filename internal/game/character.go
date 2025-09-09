// internal/domain/game/character.go (updated)
package game

import (
	"context"
	"fmt"
	"gofigure/internal/ollama"
)

// Character in the game
type Character struct {
	Name        string   `json:"name"`
	Personality string   `json:"personality"`
	Knowledge   []string `json:"knowledge"`
	Reliable    bool     `json:"reliable"`
}

// GenerateResponse using Ollama client for character interaction
func (c Character) GenerateResponse(ctx context.Context, question string, murder Murder, ollamaClient *ollama.Client) (string, error) {
	prompt := c.buildPrompt(question, murder)
	return ollamaClient.GenerateResponse(ctx, prompt)
}

func (c Character) buildPrompt(question string, murder Murder) string {
	reliabilityNote := "You are generally truthful and helpful."
	if !c.Reliable {
		reliabilityNote = "You might hide some facts, be evasive, or provide misleading information. Stay in character."
	}

	prompt := fmt.Sprintf(`You are roleplaying as %s in a murder mystery game.

CHARACTER PROFILE:
- Name: %s
- Personality: %s
- %s

MURDER SCENARIO:
- Victim found in: %s
- Murder weapon: %s  
- Actual killer: %s
- Your knowledge about the case: %v

INSTRUCTIONS:
- Stay completely in character
- Answer the detective's question based on your personality and knowledge
- Keep responses concise but engaging
- Don't break character or mention this is a game
- If you don't know something, say so in character

Detective's question: "%s"

Your response as %s:`,
		c.Name, c.Name, c.Personality, reliabilityNote,
		murder.Location, murder.Weapon, murder.Killer, c.Knowledge,
		question, c.Name)

	return prompt
}
