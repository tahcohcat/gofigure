package audio

import (
	"log"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

var musicCtrl *beep.Ctrl

var mixer = &beep.Mixer{}

func InitAudio(sampleRate beep.SampleRate) {
	speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	speaker.Play(mixer) // play everything from mixer
}

// Background music
func PlayMusic(streamer beep.Streamer) {
	mixer.Add(streamer)
}

// TTS (when audio ready)
func PlayTTS(streamer beep.Streamer) {
	mixer.Add(streamer)
}

// PlayBackgroundMusic loads and plays an MP3 in a loop.
func PlayBackgroundMusic(path string, volume float64) {
	f, err := os.Open(path)
	if err != nil {
		log.Println("Error opening mp3:", err)
		return
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Println("Error decoding mp3:", err)
		return
	}

	// Init audio system (only once)
	InitAudio(format.SampleRate)

	// Loop indefinitely
	looped := beep.Loop(-1, streamer)

	// Control for pause/resume
	musicCtrl = &beep.Ctrl{Streamer: looped, Paused: false}

	// Apply volume
	vol := &effects.Volume{
		Streamer: musicCtrl,
		Base:     2,
		Volume:   volume,
		Silent:   false,
	}

	mixer.Add(vol)
}

// PauseMusic pauses/resumes playback
func PauseMusic(pause bool) {
	if musicCtrl != nil {
		musicCtrl.Paused = pause
	}
}
