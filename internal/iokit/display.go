package iokit

import (
	"strconv"
	"strings"
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
func DisplayPath(p string) string {
	if strings.IndexFunc(p, func(r rune) bool { return !strconv.IsPrint(r) }) < 0 {
		return p
	}
	return strconv.Quote(p)
}
