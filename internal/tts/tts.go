package tts

import "context"

type Tts interface {
	Speak(ctx context.Context, text, emotions, model string) error
	Name() string
}
