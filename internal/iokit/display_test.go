package iokit

import "testing"

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
