package postgresosm

import "testing"

func TestParseYesNo(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
		ok       bool
	}{
		{"yes", true, true},
		{"No", false, true},
		{"1", true, true},
		{"0", false, true},
		{"maybe", false, false},
	}

	for _, tt := range tests {
		got, ok := parseYesNo(tt.value)
		if ok != tt.ok {
			t.Fatalf("parseYesNo(%s) ok expected %v, got %v", tt.value, tt.ok, ok)
		}
		if ok && got != tt.expected {
			t.Fatalf("parseYesNo(%s) expected %v, got %v", tt.value, tt.expected, got)
		}
	}
}

func TestEnsureName(t *testing.T) {
	if ensureName("", "amenity", 1) == "" {
		t.Fatalf("ensureName should fallback for empty name")
	}

	value := ensureName("Cafe", "amenity", 2)
	if value != "Cafe" {
		t.Fatalf("ensureName should preserve explicit name")
	}
}

func TestHashCategoryDeterministic(t *testing.T) {
	a := hashCategory("tourism")
	b := hashCategory("tourism")
	c := hashCategory("amenity")

	if a != b {
		t.Fatalf("hashCategory should be deterministic")
	}
	if a == c {
		t.Fatalf("hashCategory should differ for different inputs")
	}
}
