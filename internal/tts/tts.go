package tts

import "context"

type Tts interface {
	Speak(ctx context.Context, text, model string) error
	Name() string
}