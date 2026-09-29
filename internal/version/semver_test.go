package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		valid   bool
		major   int
		minor   int
		patch   int
		pre     string
	}{
		{"1.0.0", true, 1, 0, 0, ""},
		{"v1.2.3", true, 1, 2, 3, ""},
		{"V2.10.5", true, 2, 10, 5, ""},
		{"1.0.0-rc1", true, 1, 0, 0, "rc1"},
		{"v1.5", true, 1, 5, 0, ""},
		{"invalid", false, 0, 0, 0, ""},
		{"", false, 0, 0, 0, ""},
	}

	for _, tt := range tests {
		sv, ok := Parse(tt.input)
		if ok != tt.valid {
			t.Errorf("Parse(%q) validity = %v, expected %v", tt.input, ok, tt.valid)
			continue
		}
		if ok {
			if sv.Major != tt.major || sv.Minor != tt.minor || sv.Patch != tt.patch || sv.Pre != tt.pre {
				t.Errorf("Parse(%q) = %+v, expected major:%d minor:%d patch:%d pre:%q",
					tt.input, sv, tt.major, tt.minor, tt.patch, tt.pre)
			}
		}
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.0", "1.0.0-beta", 1},
		{"1.0.0-beta", "1.0.0", -1},
		{"v1.0.5", "v1.0.4", 1},
	}

	for _, tt := range tests {
		got := Compare(tt.v1, tt.v2)
		if got != tt.expected {
			t.Errorf("Compare(%q, %q) = %d, expected %d", tt.v1, tt.v2, got, tt.expected)
		}
	}
}

func TestIsNewer(t *testing.T) {
	if !IsNewer("1.0.1", "1.0.0") {
		t.Errorf("expected 1.0.1 to be newer than 1.0.0")
	}
	if !IsNewer("v1.1.0", "1.0.0") {
		t.Errorf("expected v1.1.0 to be newer than 1.0.0")
	}
	if IsNewer("1.0.0", "1.0.0") {
		t.Errorf("expected 1.0.0 not to be newer than 1.0.0")
	}
	if IsNewer("1.0.0", "1.0.1") {
		t.Errorf("expected 1.0.0 not to be newer than 1.0.1")
	}
}
