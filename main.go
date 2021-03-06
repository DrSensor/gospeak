/*
  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/
package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

const (
	stdSampleRate beep.SampleRate = 44100 // google=24000
	reqUrl                        = "https://simplytranslate.org/api/tts/?engine=google&lang=auto&text="
)

var src, dst string

func main() {
	close := InitUserLog("gospeak", 0.1)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	defer close()

	var text string
	var sentence <-chan *string
	var count *Statistics

	switch {
	case !term.IsTerminal(fdStdin):
		bytes, _ := io.ReadAll(os.Stdin)
		text = string(bytes[:len(bytes)-1]) // exclude carriage return
		src = "stdin"
	case len(os.Args) != 1:
		text = strings.Join(os.Args[1:], " ")
		src = "args"
	default:
		sentence, count = startTypingMode() // enter interactive/repl mode
		src = "repl"
	}

	switch err := speaker.Init(stdSampleRate, stdSampleRate.N(time.Second/10)); {
	case err != nil:
		log.Fatal().Err(err).Send()
	case sentence != nil:
		for text := <-sentence; text != nil; text = <-sentence {
			<-Speak(text)
		}
	case text != "":
		count = Stats(text)
		if done := Speak(&text); done != nil {
			<-done
		}
	default:
		log.Fatal().Msg("can't speak <empty string>")
	}

	{
		log := log.Info().
			Str("src", src).
			Object("count", count)
		if dst != "" {
			log.Str("dst", dst)
		}
		log.Msg("exit")
	}
}

var (
	fdStdin  = int(os.Stdin.Fd())
	fdStdout = int(os.Stdout.Fd())
)

func startTypingMode() (<-chan *string, *Statistics) {
	screen := Screen(os.Stdout)
	screen.EraseLine()
	oldState, err := term.MakeRaw(fdStdin)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	punc := make(chan byte)
	text := make(chan *string)
	read := Sentencing(os.Stdin, punc)
	log.Print("<~ START ~>")
	screen.SaveCursor()
	go io.Copy(screen, read)
	go func() {
		for {
			switch char := <-punc; {
			case char < 3:
				screen.Reset()
				if err := term.Restore(fdStdin, oldState); err != nil {
					log.Fatal().Err(err).Send()
				}
				log.Printf("~> CLOSE <~ ('%s' [%d])", string(char), char)
				text <- nil
			case char == 13:
				fallthrough
			default:
				log.Printf("-> ('%s' [%d])", string(char), char)
				log.Print(read.Sentences)
				text <- read.Paragraph()
				read.Sentences = &[]string{""}
			}
		}
	}()
	return text, read.count
}

const (
	kHz beep.SampleRate = 1000
	Hz  beep.SampleRate = 1
)

func Speak(text *string) (done <-chan struct{}) {
	log.Info().
		Str("engine", "google translate").
		Str("service", reqUrl).
		Msg("convert text to speech audio")
	req := reqUrl + url.QueryEscape(*text)
	switch resp, err := http.Get(req); {
	case err != nil:
		log.Err(err).Stack().Str("url", req).Send()
	case !term.IsTerminal(fdStdout) && strings.Join(os.Args[1:], " ") != "!":
		io.Copy(os.Stdout, resp.Body)
		dst = "stdout"
	default:
		done = Play(resp.Body, 4*kHz)
	}
	return
}

func Play(input io.ReadCloser, offset beep.SampleRate) <-chan struct{} {
	speaker.Clear()
	var streamer beep.Streamer // hey Go! can we have decent syntax for this
	streamer, format, err := mp3.Decode(input)
	if err != nil {
		log.Fatal().Err(err).Send()
	} else {
		log.Info().
			Int("precision", format.Precision).
			Int("numChannels", format.NumChannels).
			Dur("sampleRate", time.Duration(format.SampleRate)).
			Dur("offset", time.Duration(offset)).
			Msg("play audio!")
	}
	done := make(chan struct{})
	streamer = beep.Resample(1, format.SampleRate+offset, stdSampleRate, streamer)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() { done <- struct{}{} })))
	return done
}
