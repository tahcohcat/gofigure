package game

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gofigure/config"
	"gofigure/internal/game/audio"
	llmpkg "gofigure/internal/llm"
	"gofigure/internal/logger"
	"gofigure/internal/sst"
	"gofigure/internal/tts"
	"os"
	"strings"
	"time"
)

type Engine struct {
	murder Murder

	tts    tts.Tts
	sst    sst.Sst
	llm    llmpkg.LLM
	logger *logger.Log
	config *config.Config

	showResponses bool
	useMicInput   bool
	showHints     bool

	scanner *bufio.Scanner
}

func NewEngine(cfg *config.Config) (*Engine, error) {

	logger.New().Info(fmt.Sprintf("config.openai.apikey:%s", cfg.OpenAI.APIKey))

	llmClient, err := llmpkg.NewLLMClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
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

	// play background music
	audio.PlayBackgroundMusic("data/audio/Ketsa - Full Circles.mp3", -6)

	return &Engine{
		tts:           t,
		sst:           s,
		llm:           llmClient,
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

	if err := e.llm.IsModelAvailable(ctx); err != nil {
		e.logger.WithError(err).Error("llm model check failed")
		return fmt.Errorf("llm setup error: %w", err)
	}

	e.logger.Debug("llm connection verified")

	welcomeMessage := fmt.Sprintf("Welcome Detective! You are investigating: %s", e.murder.Title)
	e.logger.Info(fmt.Sprintf("🔍 %s", welcomeMessage))

	// Read the introduction aloud if TTS is enabled and narrator TTS model is configured
	if e.config.Tts.Enabled && len(e.murder.NarratorTTS) > 0 {
		narratorModel := e.findNarratorTtsModel()
		if narratorModel != "" {
			e.speakInterruptibleIntroduction(welcomeMessage, narratorModel)
		}
	}

	e.logger.Info(e.murder.Intro)

	return e.gameLoop()
}

func (e *Engine) gameLoop() error {

	if e.useMicInput {
		e.logger.Info("🎙️ Microphone input enabled for interviews!")
	} else {
		e.logger.Info("Type 'help' for available commands.")
		e.scanner = bufio.NewScanner(os.Stdin)
	}

	for {
		prompt := e.getPrompt()
		parts := strings.SplitN(prompt, " ", 2)

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
			e.interviewCharacter(parts[1])

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
}

func (e *Engine) showHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("  help                           - Show this help message")
	fmt.Println("  list                           - List all characters")
	fmt.Println("  interview <character>          - Interview a character")
	fmt.Println("  accuse <name> <weapon> <location> - Make your final accusation")
	fmt.Println("  quit/exit                      - Exit the game")

	if e.useMicInput {
		fmt.Println("\n🎙️ Voice Mode Enabled:")
		fmt.Println("  • Interviews automatically use voice input")
		fmt.Println("  • Press ENTER to record questions")
		fmt.Println("  • Type 'text' during interviews to switch to typing")
		fmt.Println("  • Type 'voice' during text mode to switch back")

	}
}

func (e *Engine) listCharacters() {
	fmt.Println("\nCharacters in this mystery:")
	for _, char := range e.murder.Characters {
		fmt.Printf("  • %s (%s)\n", char.Name, char.Personality)
	}
	fmt.Println()
}

func (e *Engine) interviewCharacter(charName string) {
	char := e.findCharacter(charName)
	if char == nil {
		fmt.Printf("No character named '%s' found. Enter 'list' command to see available characters.\n", charName)
		return
	}

	fmt.Printf("\n🎭 You are now interviewing %s\n", char.Name)

	if e.showHints {
		fmt.Printf("Personality: %s\n", char.Personality)
	}

	e.startInterview(char)
}

func (e *Engine) getPrompt() string {

	// mic prompt
	if e.useMicInput {
		prompt, err := e.getVoiceInput()
		if err != nil {
			fmt.Printf("Voice input failed: %v\n", err)
			return ""
		}
		s := strings.TrimSpace(strings.ToLower(prompt))
		fmt.Printf("[captured] %s\n", s)
		return s
	}

	// text prompt
	for {
		fmt.Print("> ")
		if !e.scanner.Scan() {
			break
		}

		return strings.TrimSpace(strings.ToLower(e.scanner.Text()))
	}

	return ""
}

func (e *Engine) startInterview(char *Character) {

	for {
		prompt := e.getPrompt()

		if prompt == "exit" || prompt == "quit" {
			fmt.Println("Interview ended")
			break
		}
		if prompt == "" {
			continue
		}

		e.processQuestion(char, prompt)
	}
}

func (e *Engine) processQuestion(char *Character, question string) {
	e.logger.Debug("🤔 Thinking...")

	// Use different timeout based on LLM provider
	timeout := time.Duration(e.config.Ollama.Timeout) * time.Second
	if e.config.LLM.Provider == "openai" {
		timeout = time.Duration(e.config.OpenAI.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	answer, err := char.AskQuestion(ctx, question, e.murder, e.llm)
	cancel()

	if err != nil {
		e.logger.WithError(err).Error("Failed to get character response")
		fmt.Printf("\n%s seems distracted and doesn't respond clearly.\n", char.Name)
		return
	}

	ctx, _ = context.WithTimeout(context.Background(),
		time.Duration(e.config.Ollama.Timeout)*time.Second)

	if err = e.tts.Speak(ctx, answer.Response, answer.Emotion, e.findTtsModel(char)); err != nil {
		logger.New().WithError(err).Error("character has lost their voice")
	}

	if !e.useMicInput || e.showResponses {
		e.logger.Character(char.Name, fmt.Sprintf("\r%s: [emotion:%s] %s\n", char.Name, answer.Emotion, answer.Response))
	}
}

func (e *Engine) getVoiceInput() (string, error) {
	fmt.Println("🎙️ Press ENTER to start recording...")
	fmt.Scanln()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log := logger.New()
	transcriptChan, err := e.sst.StartListening(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start listening: %w", err)
	}

	fmt.Println("🔴 Recording... Press ENTER to stop")

	// Channel to signal when user wants to stop
	stopChan := make(chan bool, 1)
	go func() {
		fmt.Scanln()
		logger.New().Debug("recording stop pressed. stopping sst and voice listener")
		err := e.sst.StopListening()
		if err != nil {
			logger.New().WithError(err).Error("failed to stop listening")
		}
		stopChan <- true
	}()

	count := 0
	// For Google SST, we need to manually process the audio chunk
	if googleSST, ok := e.sst.(*sst.GoogleSST); ok {
		go func() {
			log.Debug("[voice] initial delay for sst collection")
			time.Sleep(200 * time.Millisecond) // Longer initial delay to collect more audio

			processingInterval := 2 * time.Second // Process chunks every 2 seconds instead of 100ms
			ticker := time.NewTicker(processingInterval)
			defer ticker.Stop()

			for e.sst.IsListening() {
				select {
				case <-ticker.C:
					count++

					// Get debug stats before processing
					if debugStats := googleSST.GetDebugStats(); debugStats != nil {
						log.Debug(fmt.Sprintf("[voice] pre-processing stats [chunk:%d]: %+v", count, debugStats))
					}

					log.Debug(fmt.Sprintf("[voice] processing audio chunk [chunk:%d]", count))

					// Create a new context for each audio processing request
					chunkCtx, chunkCancel := context.WithTimeout(context.Background(), 15*time.Second)
					err := googleSST.ProcessAudioChunk(chunkCtx)
					chunkCancel()

					if err != nil {
						log.WithError(err).Error(fmt.Sprintf("failed to process audio chunk [chunk:%d]", count))
					} else {
						log.Debug(fmt.Sprintf("[voice] successfully processed chunk [chunk:%d]", count))
					}

				case <-time.After(100 * time.Millisecond):
					// Check if we should stop listening
					if !e.sst.IsListening() {
						log.Debug("[voice] sst no longer listening, exiting processing loop")
						return
					}
				}
			}

			log.Debug("[voice] sst no longer listening. processing loop exited")
		}()
	}

	var transcripts []string
	var prompt string

	// Wait for transcript, user stop, or timeout
	for {
		select {
		case transcript := <-transcriptChan:
			if strings.TrimSpace(transcript) != "" {
				transcripts = append(transcripts, strings.TrimSpace(transcript))
				// Rebuild prompt from all transcripts in order
				prompt = strings.Join(transcripts, " ")
				logger.New().Debug(fmt.Sprintf("[engine] transcript received. transcripts so far: %v, combined prompt: [%s]", transcripts, prompt))
			}
			// Continue listening for more transcripts instead of returning immediately

		case <-stopChan:
			logger.New().Debug("[engine] user requested stop")

			// Give a small delay to allow any final audio processing
			logger.New().Debug("[engine] waiting for final audio processing...")
			time.Sleep(500 * time.Millisecond)

			// Process any final transcripts that might have come in
			finalTranscriptTimeout := time.After(2 * time.Second)
			for {
				select {
				case transcript := <-transcriptChan:
					if strings.TrimSpace(transcript) != "" {
						transcripts = append(transcripts, strings.TrimSpace(transcript))
						prompt = strings.Join(transcripts, " ")
						prompt = strings.ReplaceAll(prompt, ".", " ")
						logger.New().Debug(fmt.Sprintf("[engine] final transcript received: '%s', combined: [%s]", transcript, prompt))
					}
				case <-finalTranscriptTimeout:
					logger.New().Debug("[engine] final transcript collection timeout")
					prompt = strings.ReplaceAll(prompt, ".", " ")
					return strings.TrimSpace(prompt), nil
				}
			}

		case <-ctx.Done():
			logger.New().Debug("[engine] context timeout")
			err = e.sst.StopListening()
			if err != nil {
				logger.New().WithError(err).Error("failed to stop listening on timeout")
			}

			if len(prompt) > 0 {
				prompt = strings.ReplaceAll(prompt, ".", " ")
				return strings.TrimSpace(prompt), nil
			}
			return "", fmt.Errorf("voice input timed out")
		}
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

func (e *Engine) speakInterruptibleIntroduction(welcomeMessage, narratorModel string) {
	fmt.Println("🎬 Press ENTER to skip narration, or wait to listen...")

	// Create a channel to signal if user wants to skip
	skipChan := make(chan bool, 1)

	// Goroutine to listen for user input
	go func() {
		fmt.Scanln()
		skipChan <- true
	}()

	// Start with welcome message
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(e.config.Ollama.Timeout)*time.Second)
	defer cancel()

	// Create channels for TTS completion
	ttsDone := make(chan bool, 2)

	// Speak welcome message
	go func() {
		emotion := "Welcoming and friendly"
		if err := e.tts.Speak(ctx, welcomeMessage, emotion, narratorModel); err != nil {
			if !errors.Is(err, context.Canceled) {
				e.logger.WithError(err).Error("failed to speak welcome message")
			}
		}
		ttsDone <- true
	}()

	// Wait for either user skip or TTS completion
	select {
	case <-skipChan:
		fmt.Println("🔇 Narration skipped. Let's begin the investigation!")
		return
	case <-ttsDone:
		// Welcome message finished, check if user wants to skip intro
	}

	// Check again for skip before intro
	select {
	case <-skipChan:
		fmt.Println("🔇 Narration skipped. Let's begin the investigation!")
		return
	default:
		// Continue with introduction
	}

	// Speak the introduction
	go func() {
		emotion := "Authorative, calm with a tone of mischief"
		if err := e.tts.Speak(ctx, e.murder.Intro, emotion, narratorModel); err != nil {
			e.logger.WithError(err).Error("failed to speak introduction")
		}
		ttsDone <- true
	}()

	// Wait for either user skip or intro completion
	select {
	case <-skipChan:
		fmt.Println("🔇 Narration skipped. Let's begin the investigation!")
		return
	case <-ttsDone:
		fmt.Println("🎬 Narration complete. The investigation begins!")
	}
}

func (e *Engine) findCharacter(name string) *Character {

	name = strings.ToLower(name)
	name = strings.TrimSpace(name)
	logger.New().Debug(fmt.Sprintf("[engine] finding character '%s'", name))

	for i := range e.murder.Characters {
		if strings.Contains(strings.ToLower(e.murder.Characters[i].Name), name) {
			return &e.murder.Characters[i]
		}
	}

	// try individual words - exact match
	words := strings.Split(name, " ")
	for _, word := range words {
		for i := range e.murder.Characters {
			if strings.Contains(strings.ToLower(e.murder.Characters[i].Name), word) {
				return &e.murder.Characters[i]
			}
		}
	}

	// try individual words - closest
	for _, word := range words {
		for i := range e.murder.Characters {
			n := e.murder.closesCharacterMatches().Closest(word)
			if strings.Contains(strings.ToLower(e.murder.Characters[i].Name), strings.ToLower(n)) {
				return &e.murder.Characters[i]
			}
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
