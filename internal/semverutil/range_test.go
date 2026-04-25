package semverutil

import (
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		version string
		req     string
		want    bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.0.1", "1.0.0", false},
		{"1.0.0", "[1.0.0, 2.0.0)", true},
		{"1.5.0", "[1.0.0, 2.0.0)", true},
		{"2.0.0", "[1.0.0, 2.0.0)", false},
		{"1.9.9", "(, 2.0.0]", true},
		{"2.0.0", "(, 2.0.0]", true},
		{"2.0.1", "(, 2.0.0]", false},
		{"1.0.0", "(, )", true},
		{"99.0.0", "(, )", true},
		{"1.0.0", "(1.0.0, )", false},
		{"1.0.1", "(1.0.0, )", true},
		{"1.0.0", "0.5.0", false},
		{"1.0.0", "(, 1.0.0), [2.0.0, 3.0.0), 4.0.0", false}, // 1.0.0 不在任何区间（(,1.0.0)是 <1.0.0）
		{"0.9.0", "(, 1.0.0), [2.0.0, 3.0.0), 4.0.0", true},  // 0.9.0 在 (,1.0.0)
		{"2.5.0", "(, 1.0.0), [2.0.0, 3.0.0), 4.0.0", true},  // 2.5.0 在 [2.0.0,3.0.0)
		{"4.0.0", "(, 1.0.0), [2.0.0, 3.0.0), 4.0.0", true},  // 4.0.0 精确匹配
		{"0.0.1", "[1.0.0, 2.0.0)", false},
	}

	for _, tt := range tests {
		got, err := Check(tt.version, tt.req)
		if err != nil {
			t.Errorf("Check(%q, %q) unexpected error: %v", tt.version, tt.req, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Check(%q, %q) = %v, want %v", tt.version, tt.req, got, tt.want)
		}
	}
}

func TestBestMatch(t *testing.T) {
	available := []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0", "2.1.0"}

	tests := []struct {
		req  string
		want string
	}{
		{"[1.0.0, 2.0.0)", "1.2.0"},
		{"[2.0.0, 3.0.0)", "2.1.0"},
		{">=1.1.0", ""}, // not supported format
		{"(, 2.0.0]", "2.0.0"},
		{"1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		got, err := BestMatch(available, tt.req)
		if tt.want == "" && err != nil {
			continue // expected error
		}
		if err != nil {
			t.Errorf("BestMatch(%v, %q) unexpected error: %v", available, tt.req, err)
			continue
		}
		if got != tt.want {
			t.Errorf("BestMatch(%v, %q) = %q, want %q", available, tt.req, got, tt.want)
		}
	}
}

func TestSort(t *testing.T) {
	versions := []string{"2.0.0", "1.0.0", "1.5.0", "10.0.0", "1.5.1"}
	Sort(versions)
	expected := []string{"1.0.0", "1.5.0", "1.5.1", "2.0.0", "10.0.0"}
	for i, v := range versions {
		if v != expected[i] {
			t.Errorf("Sort result[%d] = %q, want %q", i, v, expected[i])
		}
	}
}
