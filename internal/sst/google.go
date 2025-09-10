package sst

import (
	"context"
	"fmt"
	"gofigure/internal/logger"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gen2brain/malgo"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

type GoogleSST struct {
	client       *speech.Client
	languageCode string
	sampleRate   int

	// Audio capture components
	malgoContext *malgo.AllocatedContext
	device       *malgo.Device
	audioBuffer  []byte
	recording    bool

	// Add these for debugging
	totalAudioCaptured int64
	chunksProcessed    int
	lastChunkSize      int

	// Channels for communication
	transcriptChan chan string
	errorChan      chan error
}

func NewGoogleSST(ctx context.Context, languageCode string, sampleRate int) (*GoogleSST, error) {
	client, err := speech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Speech client: %w", err)
	}

	// Initialize malgo context
	malgoCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize malgo context: %w", err)
	}

	return &GoogleSST{
		client:         client,
		languageCode:   languageCode,
		sampleRate:     sampleRate,
		malgoContext:   malgoCtx,
		transcriptChan: make(chan string, 10),
		errorChan:      make(chan error, 10),
	}, nil
}

func (g *GoogleSST) StartListening(ctx context.Context) (<-chan string, error) {
	logger.New().Debug("[google-sst] start-listening called")
	if g.recording {
		return g.transcriptChan, nil
	}

	// Reset debugging counters
	g.totalAudioCaptured = 0
	g.chunksProcessed = 0
	g.lastChunkSize = 0

	// Configure audio device with better settings
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = uint32(g.sampleRate)
	deviceConfig.PeriodSizeInFrames = 1024 // Smaller buffer for more responsive capture
	deviceConfig.Periods = 4

	logger.New().Debug(fmt.Sprintf("[google-sst] audio config: format=%v, channels=%d, sampleRate=%d, periodSize=%d",
		deviceConfig.Capture.Format, deviceConfig.Capture.Channels, deviceConfig.SampleRate, deviceConfig.PeriodSizeInFrames))

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: func(outputSample, inputSample []byte, frameCount uint32) {
			if g.recording && len(inputSample) > 0 {
				g.audioBuffer = append(g.audioBuffer, inputSample...)

				// Log every 10th callback to avoid spam
				if len(g.audioBuffer)%(g.sampleRate*2) == 0 { // Every ~1 second worth of audio
					logger.New().Debug(fmt.Sprintf("[google-sst] audio captured: %d bytes total, frames in this callback: %d",
						len(g.audioBuffer), frameCount))
				}
			}
		},
	}

	device, err := malgo.InitDevice(g.malgoContext.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audio device: %w", err)
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		return nil, fmt.Errorf("failed to start audio device: %w", err)
	}

	logger.New().Debug("[google-sst] audio device started successfully")
	g.device = device
	g.recording = true

	return g.transcriptChan, nil
}

func (g *GoogleSST) StopListening() error {
	logger.New().Debug("[google-sst] stop-listening called")
	if !g.recording {
		return nil
	}

	g.recording = false

	// Process any remaining audio in the buffer before stopping
	if len(g.audioBuffer) > 0 {
		logger.New().Debug(fmt.Sprintf("[google-sst] processing final audio buffer: %d bytes", len(g.audioBuffer)))

		// Create a context for the final processing with a reasonable timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := g.ProcessAudioChunk(ctx)
		if err != nil {
			logger.New().WithError(err).Error("failed to process final audio chunk")
		} else {
			logger.New().Debug("[google-sst] final audio chunk processed successfully")
		}
	} else {
		logger.New().Debug("[google-sst] no remaining audio to process")
	}

	if g.device != nil {
		g.device.Stop()
		g.device.Uninit()
		g.device = nil
	}

	return nil
}

func (g *GoogleSST) IsListening() bool {
	return g.recording
}

func (g *GoogleSST) Provider() string {
	return "google"
}

// Add debugging method to check audio capture stats
func (g *GoogleSST) GetDebugStats() map[string]interface{} {
	return map[string]interface{}{
		"recording":           g.recording,
		"totalAudioCaptured":  g.totalAudioCaptured,
		"chunksProcessed":     g.chunksProcessed,
		"lastChunkSize":       g.lastChunkSize,
		"audioBufferSize":     len(g.audioBuffer),
		"expectedBytesPerSec": g.sampleRate * 2, // 16-bit mono
	}
}

// ProcessAudioChunk processes the current audio buffer and sends to Google STT
func (g *GoogleSST) ProcessAudioChunk(ctx context.Context) error {
	if len(g.audioBuffer) == 0 {
		logger.New().Debug("[google-sst] no audio data in buffer to process")
		return nil
	}

	// Copy buffer to avoid race conditions
	audioData := make([]byte, len(g.audioBuffer))
	copy(audioData, g.audioBuffer)
	chunkSize := len(g.audioBuffer)
	g.audioBuffer = nil // Clear buffer

	// Update debugging stats
	g.totalAudioCaptured += int64(chunkSize)
	g.chunksProcessed++
	g.lastChunkSize = chunkSize

	logger.New().Debug(fmt.Sprintf("[google-sst] processing audio chunk: size=%d bytes, total_captured=%d bytes, chunks_processed=%d",
		chunkSize, g.totalAudioCaptured, g.chunksProcessed))

	// Check if we have enough audio data (minimum ~1 second at 16kHz mono = 32KB)
	minAudioSize := g.sampleRate * 2 // 2 bytes per sample for 16-bit audio, 1 second
	if chunkSize < minAudioSize {
		logger.New().Debug(fmt.Sprintf("[google-sst] chunk too small for reliable STT: %d bytes (min: %d bytes)", chunkSize, minAudioSize))
	}

	// Send to Google STT
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz:            int32(g.sampleRate),
			LanguageCode:               g.languageCode,
			EnableAutomaticPunctuation: true,
			Model:                      "latest_long", // Better for longer phrases
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}

	logger.New().Debug("[google-sst] sending request to Google STT...")
	resp, err := g.client.Recognize(ctx, req)
	if err != nil {
		logger.New().WithError(err).Error(fmt.Sprintf("[google-sst] STT request failed for chunk %d", g.chunksProcessed))
		return err
	}

	logger.New().Debug(fmt.Sprintf("[google-sst] received %d results from Google STT", len(resp.Results)))

	for i, result := range resp.Results {
		logger.New().Debug(fmt.Sprintf("[google-sst] result %d: alternatives=%d", i, len(result.Alternatives)))

		if len(result.Alternatives) > 0 {
			transcript := result.Alternatives[0].Transcript
			confidence := result.Alternatives[0].Confidence

			logger.New().Debug(fmt.Sprintf("[google-sst] transcript: '%s' (confidence: %.2f)", transcript, confidence))

			select {
			case g.transcriptChan <- transcript:
				logger.New().Debug("[google-sst] transcript sent to channel successfully")
			case <-ctx.Done():
				logger.New().WithError(ctx.Err()).Debug("[google-sst] context cancelled while sending transcript")
				return ctx.Err()
			}
		} else {
			logger.New().Debug("[google-sst] no alternatives in result")
		}
	}

	if len(resp.Results) == 0 {
		logger.New().Debug("[google-sst] no results from Google STT - audio may be too quiet, too short, or unclear")
	}

	logger.New().Debug("[google-sst] process audio chunk finished successfully")
	return nil
}

// Close cleans up resources
func (g *GoogleSST) Close() error {
	logger.New().Debug("[google-sst] close called")

	err := g.StopListening()
	if err != nil {
		return err
	}

	if g.malgoContext != nil {
		err := g.malgoContext.Uninit()
		if err != nil {
			return err
		}
		g.malgoContext = nil
	}

	if g.client != nil {
		err := g.client.Close()
		if err != nil {
			return err
		}
		g.client = nil
	}

	close(g.transcriptChan)
	close(g.errorChan)

	return nil
}
