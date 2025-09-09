package tts

import (
	"bytes"
	"context"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	tts "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

type GoogleTTS struct {
	client *texttospeech.Client
}

func NewGoogleTTS(ctx context.Context) (*GoogleTTS, error) {
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GoogleTTS{client: client}, nil
}

func (g *GoogleTTS) Speak(ctx context.Context, text, model string) error {

	if model == "" {
		model = "en-GB-Chirp3-HD-Charon"
	}

	req := &tts.SynthesizeSpeechRequest{
		Input: &tts.SynthesisInput{
			InputSource: &tts.SynthesisInput_Text{
				Text: text,
			},

			// todo: we need to utilise this for persona and mood
			//Prompt: &text,
		},
		Voice: &tts.VoiceSelectionParams{
			LanguageCode: "en-GB",
			Name:         model,
		},
		AudioConfig: &tts.AudioConfig{
			AudioEncoding: tts.AudioEncoding_LINEAR16, // WAV PCM
		},
	}

	resp, err := g.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return err
	}

	// Decode audio in memory
	stream, format, err := wav.Decode(bytes.NewReader(resp.AudioContent))
	if err != nil {
		return err
	}
	defer stream.Close()

	// Initialize speaker
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		return err
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(stream, beep.Callback(func() {
		done <- true
	})))
	<-done
	return nil
}

func (g *GoogleTTS) Name() string {
	return "google"
}
