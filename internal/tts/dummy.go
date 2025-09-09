package tts

import (
	"context"
	"gofigure/internal/logger"
)

type DummyTts struct {
}

func NewDummyTts() *DummyTts {
	return &DummyTts{}
}

func (d *DummyTts) Speak(_ context.Context, _, _ string) error {
	logger.New().Debug("no tts configured. ignoring")
	return nil
}

func (d *DummyTts) Name() string {
	return "dummy"
}
