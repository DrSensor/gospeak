/*
  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/
package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"golang.org/x/term"
)

const (
	stdSampleRate beep.SampleRate = 44100 // google=24000
	reqUrl                        = "https://simplytranslate.org/api/tts/?engine=google&lang=auto&text="
)

func initLog(name string) func() error {
	// file, err := os.CreateTemp(name, "*.log")
	file, err := os.OpenFile(name+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	return file.Close
}

func main() {
	close := initLog("gospeak")
	defer close()

	var text string
	var sentence <-chan *string

	switch {
	case !term.IsTerminal(fdStdin):
		bytes, _ := io.ReadAll(os.Stdin)
		text = string(bytes[:len(bytes)-1]) // exclude carriage return
	case len(os.Args) != 1:
		text = strings.Join(os.Args[1:], " ")
	default:
		sentence = startTypingMode() // enter interactive/repl mode
	}

	switch err := speaker.Init(stdSampleRate, stdSampleRate.N(time.Second/10)); {
	case err != nil:
		log.Fatal(err)
	case sentence != nil:
		for {
			<-Speak(<-sentence)
		}
	case text != "":
		if done := Speak(&text); done != nil {
			<-done
		}
	default:
		log.Fatal("can't speak <empty string>")
	}
}

var (
	fdStdin  = int(os.Stdin.Fd())
	fdStdout = int(os.Stdout.Fd())
)

func startTypingMode() <-chan *string {
	screen := Screen(os.Stdout)
	screen.EraseLine()
	oldState, err := term.MakeRaw(fdStdin)
	if err != nil {
		log.Fatal(err)
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
					log.Fatal(err)
				}
				log.Printf("~> CLOSE <~ ('%s' [%d])", string(char), char)
				os.Exit(int(char))
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
	return text
}

const (
	kHz beep.SampleRate = 1000
	Hz  beep.SampleRate = 1
)

func Speak(text *string) (done <-chan struct{}) {
	req := reqUrl + url.QueryEscape(*text)
	log.Printf("request: %s", req)
	switch resp, err := http.Get(req); {
	case err != nil:
		log.Fatalf("REQUEST: %s", err)
	case !term.IsTerminal(fdStdout) && strings.Join(os.Args[1:], " ") != "!":
		io.Copy(os.Stdout, resp.Body)
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
		log.Fatalf("DECODE: %s", err)
	}
	done := make(chan struct{})
	streamer = beep.Resample(1, format.SampleRate+offset, stdSampleRate, streamer)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() { done <- struct{}{} })))
	return done
}
