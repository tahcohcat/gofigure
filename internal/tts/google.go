package tts

import (
	"bytes"
	"context"
	"fmt"
	"gofigure/internal/game/audio"
	"gofigure/internal/logger"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	tts "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"

	"github.com/faiface/beep"
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

func (g *GoogleTTS) Speak(ctx context.Context, text, emotions, model string) error {

	if model == "" {
		model = "en-GB-Chirp3-HD-Charon"
	}

	languageCode := getLanguageCode(model)

	logger.New().Debug(fmt.Sprintf("[tts] [model:%s, prompt:%s]", model, emotions))

	req := &tts.SynthesizeSpeechRequest{
		Input: &tts.SynthesisInput{
			InputSource: &tts.SynthesisInput_Text{
				Text: text,
			},

			//todo: doesn't seem well support yet..wait a bit?
			//Prompt: &emotions,
		},
		Voice: &tts.VoiceSelectionParams{
			LanguageCode: languageCode,
			Name:         model,
		},
		AudioConfig: &tts.AudioConfig{
			SampleRateHertz: 44100,
			AudioEncoding:   tts.AudioEncoding_LINEAR16, // WAV PCM
		},
	}

	resp, err := g.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return err
	}

	done := make(chan bool)

	// Decode audio in memory
	stream, _, err := wav.Decode(bytes.NewReader(resp.AudioContent))
	if err != nil {
		return err
	}
	defer stream.Close()

	ttsStream := beep.Seq(
		stream,
		beep.Callback(func() {
			done <- true
		}),
	)

	audio.PlayTTS(ttsStream)

	<-done
	return nil
}

func getLanguageCode(model string) string {
	t := strings.Split(model, "-")
	if len(t) < 3 {
		return model
	}
	return fmt.Sprintf("%s-%s", t[0], t[1])
}

func (g *GoogleTTS) Name() string {
	return "google"
}
