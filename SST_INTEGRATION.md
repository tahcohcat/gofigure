# Speech-to-Text Integration for GoFigure

This document describes the integration of Google Speech-to-Text (SST) functionality into the GoFigure murder mystery CLI game.

## Features

- **Push-to-talk voice input** during character interviews
- **Google Cloud Speech-to-Text** integration
- **Configurable language and audio settings**
- **Seamless fallback** to text input when SST is disabled or fails
- **Narrator voice** reads mystery introduction aloud when TTS is enabled

## Configuration

Add the following SST configuration to your `config.yaml`:

```yaml
sst:
  enabled: true
  provider: "google"
  language_code: "en-US"
  sample_rate: 16000
```

### Configuration Options

- `enabled`: Enable/disable SST functionality (default: false)
- `provider`: SST provider - currently only "google" is supported
- `language_code`: Language code for speech recognition (default: "en-US")
- `sample_rate`: Audio sample rate in Hz (default: 16000)

## Usage

### Enable Microphone Input

Run the game with the `--mic` flag to enable voice input:

```bash
./gofigure play data/mysteries/blackwood.json --mic
```

### During Interviews

**Streamlined Experience**: When `--mic` flag is used, interviews automatically start in voice mode:

1. **Automatic voice mode**: No need to type "mic" - voice input is default
2. **Press ENTER**: Start recording your question  
3. **Press ENTER again**: Stop recording and get transcription
4. **Seamless flow**: After the character responds, press ENTER for the next question
5. **Mode switching**: Type 'text' to switch to typing, 'voice' to switch back

### Streamlined Voice Input Flow

```
🎭 You are now interviewing Lady Blackwood
🎙️ Voice mode enabled - Press ENTER to record questions

🎙️ Press ENTER to ask a question: [Press ENTER]
🔴 Recording... Press ENTER to stop
You asked: What were you doing at the time of the murder?

[Character responds with voice]

🎙️ Press ENTER to ask a question: [Press ENTER for next question]
🔴 Recording... Press ENTER to stop  
You asked: Did you see anything suspicious?

```

## Narrator Text-to-Speech

The mystery introduction can be read aloud by a narrator voice when TTS is enabled. Add a `narrator_tts` field to your mystery JSON:

```json
{
  "title": "The Blackwood Manor Murder",
  "introduction": "It is a stormy evening at Blackwood Manor...",
  "narrator_tts": [
    {
      "engine": "google",
      "model": "en-GB-Chirp3-HD-Charon"
    }
  ],
  "characters": [...]
}
```

### Narrator Voice Experience

When you start the game, you'll hear:

1. **Welcome message** spoken by the narrator
2. **Mystery introduction** read dramatically in the narrator's voice
3. Text also appears on screen for reference

### Recommended Narrator Voices

- `en-GB-Chirp3-HD-Charon` - Distinguished British narrator
- `en-US-Chirp3-HD-Atlas` - Professional American narrator  
- `en-AU-Chirp3-HD-Titan` - Authoritative Australian narrator
- `en-GB-Wavenet-D` - Classic BBC-style narrator

## Google Cloud Setup

### Prerequisites

1. **Google Cloud Project** with Speech-to-Text API enabled
2. **Service Account** with Speech-to-Text API permissions
3. **Authentication** via one of:
   - Service account key file (set `GOOGLE_APPLICATION_CREDENTIALS`)
   - Application Default Credentials (ADC)
   - Google Cloud SDK authentication

### Authentication

Set the environment variable:
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json"
```

Or use Application Default Credentials:
```bash
gcloud auth application-default login
```

## Technical Implementation

### Architecture

- **Interface**: `sst.Sst` defines the SST contract
- **Google Implementation**: `sst.GoogleSST` handles Google Speech-to-Text
- **Dummy Implementation**: `sst.DummySST` for when SST is disabled
- **Audio Capture**: Uses `malgo` library for cross-platform audio input

### Key Components

1. **Configuration**: Extended `config.Config` with `SstConfig`
2. **CLI Integration**: Added `--mic` flag to `play` command
3. **Game Engine**: Integrated SST service with push-to-talk functionality
4. **Error Handling**: Graceful fallback to text input on SST failures

### Audio Pipeline

1. **Initialization**: Create malgo context and audio device
2. **Recording**: Capture 16kHz, 16-bit, mono audio
3. **Processing**: Send audio chunks to Google Speech-to-Text API
4. **Transcription**: Receive and display transcribed text
5. **Cleanup**: Properly dispose of audio resources

## Troubleshooting

### Common Issues

1. **No microphone detected**
   - Check system permissions
   - Verify microphone is not in use by other applications

2. **Authentication errors**
   - Verify `GOOGLE_APPLICATION_CREDENTIALS` is set
   - Check service account permissions
   - Ensure Speech-to-Text API is enabled

3. **Poor transcription quality**
   - Speak clearly and at moderate pace
   - Reduce background noise
   - Check microphone quality
   - Consider adjusting `language_code` for your locale

4. **Audio initialization failures**
   - Try different audio devices
   - Check system audio drivers
   - Verify malgo dependencies

### Debug Mode

Enable debug logging to troubleshoot issues:

```bash
export GOFIGURE_LOG_LEVEL=debug
./gofigure play mystery.json --mic
```

## Security Considerations

- **Audio data** is sent to Google Cloud for processing
- **Service account credentials** should be kept secure
- **Network traffic** contains audio and transcribed text
- Consider **data residency** requirements for your use case

## Future Enhancements

- Support for additional SST providers (Azure, AWS)
- Real-time streaming recognition
- Voice activity detection
- Audio preprocessing (noise reduction)
- Offline speech recognition options
- Custom vocabulary and language models

## Dependencies

- `cloud.google.com/go/speech` - Google Cloud Speech-to-Text
- `github.com/gen2brain/malgo` - Audio capture library
- `google.golang.org/genproto` - Google API protocol buffers

## Examples

See `cmd/test-google-sst/main.go` for a standalone SST testing example.