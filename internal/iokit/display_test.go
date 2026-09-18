package iokit

import (
	"strings"
	"testing"
)

func TestDisplayPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{"ordinary path is untouched", "/tmp/project/qode.yaml", "/tmp/project/qode.yaml"},
		{"non-ascii path is untouched", "/tmp/проект/qode.yaml", "/tmp/проект/qode.yaml"},
		{"space is printable", "/tmp/my project/qode.yaml", "/tmp/my project/qode.yaml"},
		{"ansi escape is neutralised", "/tmp/evil\x1b[2K\r/qode.yaml", `"/tmp/evil\x1b[2K\r/qode.yaml"`},
		{"newline cannot forge a line", "/tmp/x\nwarning: forged", `"/tmp/x\nwarning: forged"`},
		{"line separator cannot split a line", "/tmp/x\u2028y", "\"/tmp/x\\u2028y\""},
		{"empty path", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := DisplayPath(tc.path); got != tc.want {
				t.Errorf("DisplayPath(%q) = %s, want %s", tc.path, got, tc.want)
			}
		})
	}
}

func TestDisplayPath_EscapesRawC1Bytes(t *testing.T) {
	t.Parallel()

	// Decoding an invalid byte yields utf8.RuneError, which strconv.IsPrint calls
	// printable — so without the validity test these reach the terminal intact.
	// 0x9b is the 8-bit CSI and 0x9d the 8-bit OSC.
	tests := []struct {
		name string
		path string
	}{
		{name: "eight-bit CSI", path: "/tmp/x\x9b2Ky"},
		{name: "eight-bit OSC", path: "/tmp/x\x9d0;pwnedy"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := DisplayPath(tc.path)
			if got == tc.path {
				t.Errorf("raw C1 byte passed through unescaped: %q", got)
			}
			if strings.ContainsAny(got, "\x9b\x9d") {
				t.Errorf("escaped form still carries a raw control byte: %q", got)
			}
		})
	}
}
