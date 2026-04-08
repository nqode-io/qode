package version_test

import (
	"testing"

	"github.com/nqode/qode/internal/version"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input     string
		wantMajor int
		wantMinor int
		wantPatch int
		wantPre   string
		wantErr   bool
	}{
		{"0.1.4-alpha", 0, 1, 4, "alpha", false},
		{"v0.1.4-alpha", 0, 1, 4, "alpha", false},
		{"0.1.4-alpha+31", 0, 1, 4, "alpha", false},
		{"v0.1.4-alpha+31", 0, 1, 4, "alpha", false},
		{"0.2.0-beta", 0, 2, 0, "beta", false},
		{"1.0.0", 1, 0, 0, "", false},
		{"1.0.0-rc1", 1, 0, 0, "rc1", false},
		{"invalid", 0, 0, 0, "", true},
		{"1.2", 0, 0, 0, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			v, err := version.Parse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = nil error, want error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tc.input, err)
			}
			if v.Major != tc.wantMajor || v.Minor != tc.wantMinor || v.Patch != tc.wantPatch || v.Prerelease != tc.wantPre {
				t.Errorf("Parse(%q) = {%d,%d,%d,%q}, want {%d,%d,%d,%q}",
					tc.input, v.Major, v.Minor, v.Patch, v.Prerelease,
					tc.wantMajor, tc.wantMinor, tc.wantPatch, tc.wantPre)
			}
		})
	}
}

func TestCheckCompatibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		binary    string
		config    string
		wantError bool
	}{
		// dev builds always skip
		{"dev binary skips", "dev", "0.1.3-alpha", false},
		{"empty binary skips", "", "0.1.3-alpha", false},

		// identical versions
		{"exact match", "0.1.4-alpha", "0.1.4-alpha", false},
		{"match ignores build metadata", "0.1.4-alpha+31", "0.1.4-alpha", false},
		{"match ignores build metadata both sides", "0.1.4-alpha+31", "0.1.4-alpha+30", false},

		// alpha: patch is breaking
		{"alpha patch newer", "0.1.4-alpha", "0.1.3-alpha", true},
		{"alpha patch older", "0.1.3-alpha", "0.1.4-alpha", true},
		{"alpha minor change", "0.2.0-alpha", "0.1.4-alpha", true},
		{"alpha major change", "1.0.0-alpha", "0.1.4-alpha", true},

		// beta: patch is NOT breaking, minor IS breaking
		{"beta same minor different patch", "0.2.1-beta", "0.2.0-beta", false},
		{"beta minor change", "0.3.0-beta", "0.2.0-beta", true},
		{"beta major change", "1.0.0-beta", "0.2.0-beta", true},

		// GA / other prerelease: only major is breaking
		{"ga same major minor change", "1.1.0", "1.0.0", false},
		{"ga same major patch change", "1.0.1", "1.0.0", false},
		{"ga major change", "2.0.0", "1.0.0", true},
		{"rc same major minor change", "1.1.0-rc1", "1.0.0-rc1", false},
		{"rc major change", "2.0.0-rc1", "1.0.0-rc1", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := version.CheckCompatibility(tc.binary, tc.config)
			if tc.wantError && err == nil {
				t.Errorf("CheckCompatibility(%q, %q) = nil, want error", tc.binary, tc.config)
			}
			if !tc.wantError && err != nil {
				t.Errorf("CheckCompatibility(%q, %q) = %v, want nil", tc.binary, tc.config, err)
			}
		})
	}
}
