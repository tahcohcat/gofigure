# 🔍 GoFigure - The Voice-Enabled Murder Mystery CLI Game

**Where llamas keep the darkest secrets... and now they can hear yours too! 🦙🎙️**

Welcome to GoFigure, the most immersive murder mystery experience in your terminal! Powered by Ollama AI for intelligent character interactions and now featuring **Google Speech-to-Text** for voice-enabled interviews.

## 🎭 What is GoFigure?

GoFigure transforms your terminal into a noir detective story where you interrogate AI-powered suspects using either your keyboard *or your voice*. Each character has their own personality, secrets, and motivations - and they might just lie to your face!

## ✨ Features

### 🧠 **AI-Powered Characters**
- Dynamic conversations with Ollama-powered suspects
- Each character has unique personalities and knowledge
- Some characters are reliable, others... not so much

### 🎙️ **Voice-Enabled Interviews** *(NEW!)*
- **Push-to-talk** functionality during character interviews
- Powered by Google Cloud Speech-to-Text
- Seamlessly switch between typing and speaking
- Support for multiple languages

### 🎵 **Text-to-Speech Responses**
- Characters respond with realistic voices
- Google Cloud TTS integration
- Different voice models for different characters
- **Narrator voice** reads the mystery introduction aloud

### 🕵️ **Interactive Mystery Solving**
- Interview suspects to gather clues
- Make accusations when you think you've solved it
- **Two compelling mysteries** with different settings and complexity
- JSON-based mystery scenarios (easily customizable!)

### 🎪 **Available Mysteries**
- **🏰 The Blackwood Manor Murder** - Classic English manor mystery with aristocratic intrigue
- **🚢 Death on the Aurora Star** - Complex cruise ship thriller with international suspects and red herrings

## 🚀 Quick Start

### Prerequisites

1. **Ollama** running locally with a model (e.g., `llama3.2`)
2. **Google Cloud credentials** (optional, for voice features)
3. **Go 1.23+**

### Installation

```bash
# Clone the repository
git clone https://github.com/tahcohcat/gofigure.git
cd gofigure

# Build the game
go build -o gofigure ./cmd/gofigure

# Play a mystery with voice enabled!
./gofigure play data/mysteries/cruise_ship.json --mic
```

## 🎮 How to Play

### Basic Commands

```bash
# Play the classic manor mystery
./gofigure play data/mysteries/blackwood.json

# Try the cruise ship thriller (with twists!)
./gofigure play data/mysteries/cruise_ship.json --mic

# Check your configuration
./gofigure config
```

### In-Game Commands

- `help` - Show available commands
- `list` - List all characters in the mystery
- `interview <character>` - Start questioning a suspect
- `accuse <name> <weapon> <location>` - Make your final accusation
- `exit` - Quit the game


### 🎙️ Streamlined Voice Input

When you start an interview with `--mic` enabled:

- **Automatic voice mode** - No need to type "mic" every time!
- **Press ENTER** to record each question
- **Seamless flow** - Ask questions naturally with your voice
- **Switch modes** - Type 'text' to switch to typing, 'voice' to switch back

```
🎭 You are now interviewing Lady Blackwood
🎙️ Voice mode enabled - Press ENTER to record questions

🎙️ Press ENTER to ask a question: [Press ENTER]


```
Q: mic
🎙️ Press ENTER to start recording...

🔴 Recording... Press ENTER to stop
You asked: What were you doing at the time of the murder?

🎭 Character: Well, detective, I was in the library reading...


🎙️ Press ENTER to ask a question: [Press ENTER again for next question]

```

### 🎙️ Immersive Audio Experience

When TTS is enabled, the game provides a rich audio experience:

- **🎬 Narrator** reads the mystery introduction in a distinguished voice
- **🎭 Characters** speak their responses with unique voice models  
- **📖 Welcome messages** are announced dramatically

```
🔍 Welcome Detective! You are investigating: The Blackwood Manor Murder
[Narrator voice speaks the welcome and introduction]
```

## ⚙️ Configuration

Create a `config.yaml` file:

```yaml
ollama:
  host: "http://localhost:11434"
  model: "llama3.2"
  timeout: 50

tts:
  enabled: true
  type: "google"

sst:
  enabled: true           # Enable voice input
  provider: "google"
  language_code: "en-US"  # or "en-GB", "es-ES", etc.
  sample_rate: 16000
```

### Google Cloud Setup (for voice features)

1. Create a Google Cloud project
2. Enable Speech-to-Text and Text-to-Speech APIs
3. Create a service account with appropriate permissions
4. Download the JSON key file
5. Set the environment variable:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/credentials.json"
```

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Your Voice    │───▶│   Google STT    │───▶│  Game Engine    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Your Speakers  │◄───│   Google TTS    │◄───│  Ollama LLM     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Key Components

- **Game Engine** (`internal/game/`) - Core mystery logic
- **SST Service** (`internal/sst/`) - Speech-to-Text integration
- **TTS Service** (`internal/tts/`) - Text-to-Speech integration
- **Ollama Client** (`internal/ollama/`) - AI character conversations
- **Config System** (`config/`) - YAML-based configuration

## 🎨 Creating Your Own Mysteries

Mysteries are defined in JSON format. Here's the structure:

```json
{
  "title": "The Case of the Missing Cookies",
  "killer": "Butler",
  "weapon": "Rolling Pin",
  "location": "Kitchen",
  "introduction": "The cookies have vanished...",
  "narrator_tts": [
    {
      "engine": "google",
      "model": "en-GB-Chirp3-HD-Charon"
    }
  ],
  "characters": [
    {
      "name": "Chef Williams",
      "personality": "Anxious and defensive",
      "knowledge": [
        "I was preparing dinner when the cookies disappeared",
        "The butler seemed suspicious lately"
      ],
      "reliable": true,
      "tts": [
        {
          "engine": "google",
          "model": "en-US-Chirp3-HD-Mason"
        }
      ]
    }
  ]
}
```

## 🛠️ Development

### Project Structure

```
gofigure/
├── cmd/
│   ├── gofigure/           # Main application
│   └── test-google-sst/    # SST testing tool
├── internal/
│   ├── game/              # Core game logic
│   ├── sst/               # Speech-to-Text
│   ├── tts/               # Text-to-Speech
│   ├── ollama/            # Ollama integration
│   └── logger/            # Logging utilities
├── config/                # Configuration management
├── data/mysteries/        # Mystery scenario files
└── docs/                  # Documentation
```

### Building from Source

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o gofigure ./cmd/gofigure

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o gofigure-linux ./cmd/gofigure
GOOS=windows GOARCH=amd64 go build -o gofigure.exe ./cmd/gofigure
GOOS=darwin GOARCH=amd64 go build -o gofigure-mac ./cmd/gofigure
```

## 🎯 Troubleshooting

### Common Issues

**"No microphone detected"**
- Check system permissions
- Ensure microphone isn't used by other apps

**"Google authentication failed"**
- Verify `GOOGLE_APPLICATION_CREDENTIALS` is set
- Check service account permissions

**"Ollama connection failed"**
- Start Ollama: `ollama serve`
- Pull a model: `ollama pull llama3.2`

**"Poor speech recognition"**
- Speak clearly at moderate pace
- Reduce background noise
- Check microphone quality

### Debug Mode

```bash
export GOFIGURE_LOG_LEVEL=debug
./gofigure play mystery.json --mic --show-responses
```

## 🤝 Contributing

We welcome contributions! Here are some ideas:

- 🎭 **New mystery scenarios** - Create engaging cases
- 🌍 **Language support** - Add more language codes
- 🎤 **Audio improvements** - Noise reduction, better detection
- 🤖 **AI enhancements** - Better character personalities
- 🔧 **Platform support** - Windows/Linux testing

### Development Setup

```bash
git clone https://github.com/tahcohcat/gofigure.git
cd gofigure
go mod download
go run ./cmd/gofigure play data/mysteries/blackwood.json
```

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- **Ollama** - For making local AI accessible
- **Google Cloud** - For excellent Speech and TTS APIs
- **gen2brain/malgo** - For cross-platform audio capture
- **spf13/cobra** - For the CLI framework
- **spf13/viper** - For configuration management

## 🎪 Fun Examples

### Voice Commands During Interviews

Try these voice commands during interviews:

- *"What's your alibi for last night?"*
- *"Did you see anything suspicious?"*
- *"Where were you when the lights went out?"*
- *"Tell me about your relationship with the victim."*

### Narrator Voice Immersion

Enable TTS to experience atmospheric introductions:

**🏰 Blackwood Manor:**
🎬 *"It is a stormy evening at Blackwood Manor. The guests have gathered for Lady Blackwood's birthday celebration when a terrible cry echoes through the corridors..."*

**🚢 Aurora Star Cruise:**  
🎬 *"The luxury cruise ship Aurora Star glides through the Mediterranean under a blanket of stars. At 11:32 PM, during the height of the festivities, a blood-curdling scream pierces through the music..."*

Each mystery has its own distinct narrator setting the perfect mood!

## 🐛 Found a Bug?

Please [open an issue](https://github.com/tahcohcat/gofigure/issues) with:
- Your OS and Go version
- Steps to reproduce
- Expected vs actual behavior
- Log output (with `--show-responses` flag)

---

**Happy detecting, and may your voice be heard! 🕵️‍♀️🎙️**

*P.S. - The llamas are always watching... 🦙👀*
