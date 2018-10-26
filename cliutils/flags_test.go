package cliutils

import (
	"testing"
)

func TestFlagNSMap(t *testing.T) {
	// Flag accepts "5000" and "name:5000"
	f := FlagNSMap{}

	f.Set("5000")
	if val, ok := f["5000"]; !ok || val != 5000 {
		t.Errorf("Expected 5000 => 5000, got 5000 => %d", val)
	}

	f.Set("name:6000")
	if val, ok := f["name"]; !ok || val != 6000 {
		t.Errorf("Expected name => 6000, got name => %d", val)
	}
}
