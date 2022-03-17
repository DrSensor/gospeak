/*
  Copying and distribution of this file, with or without modification,
  are permitted in any medium without royalty provided the copyright
  notice and this notice are preserved. This file is offered as-is,
  without any warranty.
*/
package main

import "io"

// Screen must be os.Stdout, os.Stderr, or similiar
// which accept ANSI escape sequences for in-band signaling.
func Screen(screen io.Writer) *ScreenCtl {
	return &ScreenCtl{screen}
}

type ScreenCtl struct{ io.Writer }

// EraseLine usually used at program init to avoid bug caused by other program.
// (i.e tmux, shell prompt, ...)
func (screen *ScreenCtl) EraseLine() { screen.Writer.Write(tEraseLine[:]) }

// SaveCursor at current position for future use.
// (i.e Reset)
func (screen *ScreenCtl) SaveCursor() { screen.Writer.Write(tMoveCursor[:]) }

// Reset to erase content from screen until saved cursor.
// All content befor SaveCursor will not erased.
func (screen *ScreenCtl) Reset() { screen.Writer.Write(tClear[:]) }

var ( // https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797 for more details.
	tMoveCursor = [...]byte{27, '7'}
	tEraseLine  = [...]byte{27, '[', '2', 'K'}
	tClear      = [...]byte{27, '8', 27, '[', '0', 'J'}
	tNull       = [6]byte{}
)

func IsWhitespace(char byte) bool { // TODO: handle modifier key (i.e Shift, Ctrl, Alt)
	switch char {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	}
	return false
}

func IsArrow(seq *[6]byte) bool {
	return (seq[0] == 27 && seq[1] == 91) && isArrow(seq[2]) || // normal Arrow
		(seq[2] == 49 && seq[3] == 59) && (seq[4] <= 56 && seq[4] >= 50) && isArrow(seq[5]) // Arrow with modifier (i.e Shift, Ctrl, Alt)
}
func isArrow(char byte) bool { return char <= 68 && char >= 65 }