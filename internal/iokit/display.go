package iokit

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// DisplayPath renders a filesystem path for printing to a terminal, a log or an
// agent's transcript. A path is not trusted text: it comes from --root, the working
// directory or a cloned repository's own directory names, so it can carry an ANSI
// escape that erases the line it is printed on, or a newline that forges a second
// line reading like qode's own output. A path that is already printable is returned
// unchanged, so ordinary output is untouched; one that is not is quoted, which
// turns every control character into an escape such as \x1b and makes the forgery
// visible instead of effective. strconv.IsPrint rejects the Unicode line and
// paragraph separators too, so those cannot split a line either.
//
// The UTF-8 validity test is not redundant: decoding an invalid byte yields
// utf8.RuneError, which strconv.IsPrint reports as printable, so a raw 0x9b (the
// 8-bit form of CSI) or 0x9d (OSC) would otherwise reach the terminal intact. A
// filename can carry those bytes on Linux, which is the release target.
func DisplayPath(p string) string {
	if utf8.ValidString(p) && strings.IndexFunc(p, func(r rune) bool { return !strconv.IsPrint(r) }) < 0 {
		return p
	}
	return strconv.Quote(p)
}
