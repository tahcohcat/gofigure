package sst

import (
	"context"
	"fmt"
	"log"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gen2brain/malgo"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

type GoogleSST struct {
	client         *speech.Client
	languageCode   string
	sampleRate     int
	
	// Audio capture components
	malgoContext   *malgo.AllocatedContext
	device         *malgo.Device
	audioBuffer    []byte
	recording      bool
	
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
	if g.recording {
		return g.transcriptChan, nil
	}

	// Configure audio device
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = uint32(g.sampleRate)

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: func(outputSample, inputSample []byte, frameCount uint32) {
			if g.recording {
				g.audioBuffer = append(g.audioBuffer, inputSample...)
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

	g.device = device
	g.recording = true
	
	return g.transcriptChan, nil
}

func (g *GoogleSST) StopListening() error {
	if !g.recording {
		return nil
	}

	g.recording = false
	
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

// ProcessAudioChunk processes the current audio buffer and sends to Google STT
func (g *GoogleSST) ProcessAudioChunk(ctx context.Context) error {
	if len(g.audioBuffer) == 0 {
		return nil
	}

	// Copy buffer to avoid race conditions
	audioData := make([]byte, len(g.audioBuffer))
	copy(audioData, g.audioBuffer)
	g.audioBuffer = nil // Clear buffer

	// Send to Google STT
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: int32(g.sampleRate),
			LanguageCode:    g.languageCode,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}

	resp, err := g.client.Recognize(ctx, req)
	if err != nil {
		log.Printf("Error recognizing speech: %v", err)
		return err
	}

	for _, result := range resp.Results {
		if len(result.Alternatives) > 0 {
			transcript := result.Alternatives[0].Transcript
			select {
			case g.transcriptChan <- transcript:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// Close cleans up resources
func (g *GoogleSST) Close() error {
	g.StopListening()
	
	if g.malgoContext != nil {
		g.malgoContext.Uninit()
		g.malgoContext = nil
	}
	
	if g.client != nil {
		g.client.Close()
		g.client = nil
	}
	
	close(g.transcriptChan)
	close(g.errorChan)
	
	return nil
}