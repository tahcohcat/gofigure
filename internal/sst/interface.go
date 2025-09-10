package sst

import "context"

// Sst defines the interface for speech-to-text services
type Sst interface {
	// StartListening begins audio capture and returns a channel for the transcribed text
	StartListening(ctx context.Context) (<-chan string, error)
	
	// StopListening stops audio capture
	StopListening() error
	
	// IsListening returns true if currently capturing audio
	IsListening() bool
	
	// Provider returns the name of the SST provider
	Provider() string
}