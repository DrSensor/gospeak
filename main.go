package main

// TODO: 1. buffer microphone stream -> play buffer into speaker -> repeat from step-1
// TODO: 2. stream buffer into websocket

import (
	"time"

	"github.com/MarkKremer/microphone"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

const stdSampleRate beep.SampleRate = 44100

func main() {
	check(microphone.Init())
	defer try(microphone.Terminate)

	stream, format := check2(microphone.OpenDefaultStream(stdSampleRate, 2))
	defer try(stream.Close)

	check(speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)))
	// defer speaker.Close() // sometimes cause SIGSEGV on microphone.Terminate()
	defer speaker.Clear()

	check(stream.Start())
	// defer try(stream.Stop) // sometimes cause speaker.Clear() block indefinitely
	speaker.Play(stream)

	<-exit()
}
