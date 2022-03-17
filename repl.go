/*
  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/
package main

import (
	"io"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func Sentencing(r io.Reader, punctuation chan<- byte) *SentenceReader {
	return &SentenceReader{
		Reader: r, punc: punctuation,
		Sentences: &[]string{""},
	}
}

type SentenceReader struct {
	io.Reader
	Sentences *[]string
	punc      chan<- byte
}

func (s *SentenceReader) Paragraph() *string {
	p := (strings.Join(*s.Sentences, " "))
	return &p
}

func (s *SentenceReader) pointLastSenstence() *string {
	return &(*s.Sentences)[len(*s.Sentences)-1]
}

func (s *SentenceReader) queue(sentence *string) {
	cs := s.pointLastSenstence()
	if *cs != "" {
		*s.Sentences = append(*s.Sentences, "")
		cs = s.pointLastSenstence()
	}
	*cs = *sentence
	*sentence = ""
}

func isPunctuation(char byte) bool { return char == '.' || char == '!' || char == '?' || char == ',' }

func (s *SentenceReader) WriteTo(w io.Writer) (int64, error) {
	var (
		buf            = tNull
		capitalize     = true
		sentence       string
		char, charPrev byte
		timeout        *time.Timer
	)
	for { // should I follow https://github.com/golang/go/blob/master/src/io/io.go#L425-L452 🤔?
		nr, err := s.Reader.Read(buf[:])
		if err != nil {
			s.punc <- 1
			return int64(nr), err
		}
		char = buf[0]

		switch {
		case (nr == 3 || nr == 6) && IsArrow(&buf):
			buf = tNull
		case char == 13 || char == 27:
			log.Printf("^ %s [%d]", string(char), char)
			buf = tClear
		case char == 3: // Ctrl-c
			s.punc <- 0
			return int64(len(buf)), nil
		case isPunctuation(charPrev) && IsWhitespace(char):
			if timeout != nil {
				timeout.Stop()
				if charPrev != ',' {
					s.queue(&sentence)
					log.Printf("<- ('%s' [%d]) @ ('%s' [%d])", string(charPrev), charPrev, string(char), char)
					s.punc <- charPrev
				}
			}
			if charPrev != ',' {
				capitalize = true
			}
		default:
			log.Print(string(buf[:]))
			sentence += string(char)
			if capitalize {
				copy(buf[:], strings.ToTitle(string(buf[:])))
				capitalize = false
			}
		}

		switch nw, err := w.Write(buf[:]); {
		case err != nil:
			s.punc <- 2
			return int64(nw), err
		case isPunctuation(char) && !IsWhitespace(charPrev) && !isPunctuation(charPrev):
			timeout = time.AfterFunc(360*time.Millisecond, func() {
				timeout = nil
				s.queue(&sentence)
				log.Printf("<- ('%s' [%d])", string(char), char)
				s.punc <- char
			})
		case char == 13:
			buf = tNull
			s.punc <- char
		}

		charPrev = char
	}
}