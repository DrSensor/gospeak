package main

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"golang.org/x/term"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	var text string
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		if bytes, err := io.ReadAll(os.Stdin); err == nil {
			text = string(bytes[:len(bytes)-1])
		}
	} else {
		text = strings.Join(os.Args[1:], " ")
	}

	switch resp, err := http.Get("https://simplytranslate.org/api/tts/?engine=google&lang=auto&text=" + url.QueryEscape(text)); {
	case err != nil:
		log.Fatal(err)

	case !term.IsTerminal(int(os.Stdout.Fd())):
		io.Copy(os.Stdout, resp.Body)

	default:
		streamer, format, err := mp3.Decode(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		defer streamer.Close()
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		done := make(chan bool)
		speaker.Play(beep.Seq(streamer, beep.Callback(func() { done <- true })))
		<-done
	}
}
