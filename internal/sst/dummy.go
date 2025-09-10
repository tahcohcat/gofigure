package sst

import (
	"context"
)

// DummySST is a no-op implementation for when SST is disabled
type DummySST struct{}

func NewDummySST() *DummySST {
	return &DummySST{}
}

func (d *DummySST) StartListening(ctx context.Context) (<-chan string, error) {
	// Return an empty channel that will never receive data
	ch := make(chan string)
	close(ch)
	return ch, nil
}

func (d *DummySST) StopListening() error {
	return nil
}

func (d *DummySST) IsListening() bool {
	return false
}

func (d *DummySST) Provider() string {
	return "dummy"
}