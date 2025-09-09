package game

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"gofigure/config"
	"gofigure/internal/logger"
	"gofigure/internal/ollama"
	"gofigure/internal/sst"
	"gofigure/internal/tts"
	"os"
	"strings"
	"time"
)

type Engine struct {
	murder Murder

	tts          tts.Tts
	sst          sst.Sst
	ollamaClient *ollama.Client
	logger       *logger.Log
	config       *config.Config

	showResponses bool
	useMicInput   bool
}

func NewEngine(cfg *config.Config) (*Engine, error) {
	ollamaClient, err := ollama.NewClient(&cfg.Ollama)
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var t tts.Tts
	t = tts.NewDummyTts()

	if cfg.Tts.Enabled {
		t, err = tts.NewGoogleTTS(ctx)
		if err != nil {
			logger.New().WithError(err).Error("failed to create tts TTS client")
			t = tts.NewDummyTts()
		} else {
			logger.New().Debug("google tts client created")
		}
	}

	// Initialize SST
	var s sst.Sst
	s = sst.NewDummySST()

	if cfg.Sst.Enabled && cfg.Sst.Provider == "google" {
		s, err = sst.NewGoogleSST(ctx, cfg.Sst.LanguageCode, cfg.Sst.SampleRate)
		if err != nil {
			logger.New().WithError(err).Error("failed to create Google SST client, using dummy")
			s = sst.NewDummySST()
		} else {
			logger.New().Debug("google sst client created")
		}
	}

	return &Engine{
		tts:           t,
		sst:           s,
		ollamaClient:  ollamaClient,
		logger:        logger.New(),
		config:        cfg,
		showResponses: false,
		useMicInput:   true,
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

func (e *Engine) WithMicInput(useMic bool) *Engine {
	e.useMicInput = useMic && e.config.Sst.Enabled
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

	e.logger.Debug("ollama connection verified")

	welcomeMessage := fmt.Sprintf("🔍 Welcome Detective! You are investigating: %s", e.murder.Title)
	e.logger.Info(welcomeMessage)

	// Read the introduction aloud if TTS is enabled and narrator TTS model is configured
	if e.config.Tts.Enabled && len(e.murder.NarratorTTS) > 0 {
		narratorModel := e.findNarratorTtsModel()
		if narratorModel != "" {
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Duration(e.config.Ollama.Timeout)*time.Second)

			// Speak the welcome message
			if err := e.tts.Speak(ctx, welcomeMessage, narratorModel); err != nil {
				e.logger.WithError(err).Error("failed to speak welcome message")
			}

			// Speak the introduction

			//todo find a way to skip it
			//if err := e.tts.Speak(ctx, e.murder.Intro, narratorModel); err != nil {
			//	e.logger.WithError(err).Error("failed to speak introduction")
			//}

			cancel()
		}
	}

	e.logger.Info(e.murder.Intro)
	e.logger.Info("Type 'help' for available commands.")

	if e.useMicInput {
		e.logger.Info("🎙️ Microphone input enabled for interviews!")
	}

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

	if e.useMicInput {
		fmt.Println("\n🎙️ Microphone Features:")
		fmt.Println("  During interviews, you can:")
		fmt.Println("  • Type 'mic' or press ENTER to use voice input (push-to-talk)")
		fmt.Println("  • Continue typing questions normally")
	}
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

	if e.useMicInput {
		fmt.Println("💡 Tip: Type 'mic' to use voice input (push-to-talk), or continue typing normally")
	}

	for {
		fmt.Print("\nQ: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "exit" {
			fmt.Println("Interview ended.\n")
			break
		}
		if input == "" {
			continue
		}

		var question string
		var err error

		// Check if user wants to use microphone
		if (input == "mic" || input == "\n") && e.useMicInput {
			question, err = e.getVoiceInput()
			if err != nil {
				fmt.Printf("Voice input failed: %v\n", err)
				continue
			}
			if question == "" {
				fmt.Println("No speech detected, please try again.")
				continue
			}
			fmt.Printf("You asked: %s\n", question)
		} else {
			question = input
		}

		e.logger.Debug("🤔 Thinking...")

		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(e.config.Ollama.Timeout)*time.Second)

		answer, err := char.AskQuestion(ctx, question, e.murder, e.ollamaClient)
		cancel()

		if err != nil {
			e.logger.WithError(err).Error("Failed to get character response")
			fmt.Printf("\n%s seems distracted and doesn't respond clearly.\n", char.Name)
			continue
		}

		ctx, _ = context.WithTimeout(context.Background(),
			time.Duration(e.config.Ollama.Timeout)*time.Second)

		if err = e.tts.Speak(ctx, answer, e.findTtsModel(char)); err != nil {
			logger.New().WithError(err).Error("character has lost their voice")
		}

		if e.showResponses {
			e.logger.Character(char.Name, fmt.Sprintf("👨‍✈️.....\r%s: %s\n", char.Name, answer))
		}
	}
}

func (e *Engine) getVoiceInput() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("🎙️ Press ENTER to start recording...")
	fmt.Scanln()

	transcriptChan, err := e.sst.StartListening(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start listening: %w", err)
	}

	fmt.Println("🔴 Recording... Press ENTER to stop")
	go func() {
		fmt.Scanln()
		e.sst.StopListening()
	}()

	// For Google SST, we need to manually process the audio chunk
	if googleSST, ok := e.sst.(*sst.GoogleSST); ok {
		go func() {
			time.Sleep(1 * time.Second) // Give some time to collect audio
			for e.sst.IsListening() {
				googleSST.ProcessAudioChunk(ctx)
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	// Wait for transcript or timeout
	select {
	case transcript := <-transcriptChan:
		e.sst.StopListening()
		return strings.TrimSpace(transcript), nil
	case <-ctx.Done():
		e.sst.StopListening()
		return "", fmt.Errorf("voice input timed out")
	}
}

func (e *Engine) findTtsModel(character *Character) string {
	for _, ttsOption := range character.TTS {
		if ttsOption.Engine == e.tts.Name() {
			return ttsOption.Model
		}
	}

	return ""
}

func (e *Engine) findNarratorTtsModel() string {
	for _, ttsOption := range e.murder.NarratorTTS {
		if ttsOption.Engine == e.tts.Name() {
			return ttsOption.Model
		}
	}

	return ""
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

func (e *Engine) WithResponses(resp bool) *Engine {
	e.showResponses = resp
	return e
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
