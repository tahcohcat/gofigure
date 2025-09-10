// internal/domain/game/character.go (updated)
package game

import (
	"context"
	"encoding/json"
	"fmt"
	"gofigure/internal/llm"
	"gofigure/internal/logger"
	"time"
)

type Message struct {
	Role      string    `json:"role,omitempty"`
	Content   string    `json:"content,omitempty" json:"content,omitempty"`
	Emotions  string    `json:"emotions,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type TTS struct {
	Engine string `json:"engine,omitempty"`
	Model  string `json:"model,omitempty"`
}

// Character in the game
type Character struct {
	Name        string   `json:"name"`
	Personality string   `json:"personality"`
	Knowledge   []string `json:"knowledge"`
	Reliable    bool     `json:"reliable"`
	TTS         []TTS    `json:"tts"`

	Conversation []*Message
}

func (c *Character) GetCharacterResponse(ctx context.Context, prompt string, llmClient llm.LLM) (*llm.CharacterReply, error) {

	resp, err := llmClient.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var reply llm.CharacterReply
	if err := json.Unmarshal([]byte(resp), &reply); err != nil {
		logger.New().Warn(fmt.Sprintf("failed to unmarshal response. [response:%s, prompt:%s]", resp, prompt))
		return nil, err
	}
	return &reply, nil
}

// AskQuestion using Ollama client for character interaction
func (c *Character) AskQuestion(ctx context.Context, question string, murder Murder, llmClient llm.LLM) (*llm.CharacterReply, error) {

	c.addQuestion(question, murder)

	prompt := c.serialiseConversation()

	resp, err := c.GetCharacterResponse(ctx, prompt, llmClient)
	if err != nil {
		logger.New().WithError(err).Warn("could not generate character response")
		return &llm.CharacterReply{}, err
	}

	c.Conversation = append(c.Conversation, &Message{

		// openai supperted types:  ['system', 'assistant', 'user', 'function', 'tool', and 'developer']",
		Role:      "assistant",
		Content:   resp.Response,
		Emotions:  resp.Emotion,
		Timestamp: time.Now(),
	})

	return resp, nil
}

func (c *Character) addQuestion(question string, murder Murder) {
	reliabilityNote := "You are generally truthful and helpful."
	if !c.Reliable {
		reliabilityNote = "You might hide some facts, be evasive, or provide misleading information. Stay in character."
	}

	latest := fmt.Sprintf("Detective's follow up question: %s", question)

	if c.IsInitialMessage() {
		scenario := fmt.Sprintf(`You are roleplaying as %s in a murder mystery game.

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
- Give the character response and derive their emotional state
- Reply in this JSON structure {"response": string, "emotion": string}

Detective's question: "%s"

Your response as %s:`,
			c.Name, c.Name, c.Personality, reliabilityNote,
			murder.Location, murder.Weapon, murder.Killer, c.Knowledge,
			question, c.Name)

		c.Conversation = []*Message{
			{Role: "system", Content: fmt.Sprintf("%s", scenario), Timestamp: time.Now()},
		}

		latest = fmt.Sprintf("Detective's question: %s", question)
	}

	c.Conversation = append(c.Conversation, &Message{Role: "user", Content: latest, Timestamp: time.Now()})
}

func (c *Character) IsInitialMessage() bool {
	return len(c.Conversation) == 0
}

func (c *Character) serialiseConversation() string {
	s, err := json.Marshal(c.Conversation)
	if err != nil {
		logger.New().Error(err.Error())
		return ""
	}

	return string(s)
}
