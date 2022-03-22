package main

import (
	"os"
	"os/signal"
)

func exit() <-chan os.Signal {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	go func() { <-sig; signal.Stop(sig); close(sig) }()
	return sig
}

func try2[T, R any](call func(a T, b R) error, a T, b R) { check(call(a, b)) }
func try1[T any](call func(a T) error, a T)              { check(call(a)) }
func try(call func() error)                              { check(call()) }

func check2[T, R any](a T, b R, err error) (T, R) { check(err); return a, b }
func check1[T any](a T, err error) T              { check(err); return a }
func check(err error) {
	if err != nil {
		panic(err)
	}
}