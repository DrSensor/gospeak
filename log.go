/*
  Copying and distribution of this file, with or without modification,
  are permitted in any medium without royalty provided the copyright
  notice and this notice are preserved. This file is offered as-is,
  without any warranty.
*/
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func slug(length int) string {
	b := make([]byte, length/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}
func createLogFile(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0666)
}

func InitUserLog(name string, version float32) func() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
	}
	dir := filepath.Join(home, logDir, name)
	os.Mkdir(dir, 0755)
	file, err := createLogFile(filepath.Join(dir, slug(8)+".log"))
	for os.IsExist(err) { // https://stefanxo.com/go-anti-patterns-os-isexisterr-os-isnotexisterr
		file, err = createLogFile(filepath.Join(dir, slug(8)+".log"))
	}
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		log.Logger = zerolog.New(io.Discard)
	} else {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = zerolog.New(file).With().
			Float32("version", version).
			Timestamp().Logger()

	}
	return func() {
		if err := file.Close(); err != nil {
			log.Fatal().Err(err).Send()
		}
	}
}

type Statistics struct {
	word     uint32
	sentence uint16
}

func (count Statistics) MarshalZerologObject(log *zerolog.Event) {
	log.Uint16("sentence", count.sentence).
		Uint32("word", count.word)
}

func Stats(text string) *Statistics {
	var charPrev byte
	count := &Statistics{}
	for index, char := range text {
		char := byte(char)
		switch {
		case (isPunctuation(charPrev) && IsWhitespace(char)) || index == len(text)-1:
			if !isComma(charPrev) {
				count.sentence++
			}
			count.word++
		case (IsWhitespace(char) && !IsWhitespace(charPrev) && !isPunctuation(charPrev)) ||
			(isHyphen(char) && !isHyphen(charPrev)):
			count.word++
		}
		charPrev = char
	}
	return count
}