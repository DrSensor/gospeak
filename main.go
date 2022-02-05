package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	resp, err := http.Get("https://simplytranslate.org/api/tts/?engine=google&lang=auto&text=" + url.QueryEscape(strings.Join(os.Args[1:], " ")))
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(os.Stdout, resp.Body)
}
