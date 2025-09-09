package main

import (
	"context"
	"fmt"
	"log"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gen2brain/malgo"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

func main() {
	// Google Speech client
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Audio buffer
	var audioBuffer []byte
	recording := false

	// Init malgo
	ctxMal, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ctxMal.Uninit()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 16000

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: func(outputSample, inputSample []byte, frameCount uint32) {
			if recording {
				audioBuffer = append(audioBuffer, inputSample...)
			}
		},
	}

	device, err := malgo.InitDevice(ctxMal.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Push-to-talk demo (press ENTER to start recording, ENTER again to stop)")

	for {
		fmt.Println("\nPress ENTER to start recording a chunk...")
		fmt.Scanln()
		audioBuffer = nil
		recording = true
		fmt.Println("Recording... press ENTER to stop")
		fmt.Scanln()
		recording = false

		// Send audioBuffer to Google STT
		req := &speechpb.RecognizeRequest{
			Config: &speechpb.RecognitionConfig{
				Encoding:        speechpb.RecognitionConfig_LINEAR16,
				SampleRateHertz: 16000,
				LanguageCode:    "en-US",
			},
			Audio: &speechpb.RecognitionAudio{
				AudioSource: &speechpb.RecognitionAudio_Content{
					Content: audioBuffer,
				},
			},
		}

		resp, err := client.Recognize(ctx, req)
		if err != nil {
			log.Println("Error sending audio:", err)
			continue
		}

		for _, result := range resp.Results {
			fmt.Printf("Transcript: %s\n", result.Alternatives[0].Transcript)
		}
	}
}
