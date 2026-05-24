package handlers

import "testing"

func TestLoadTemplates(t *testing.T) {
	if err := LoadTemplates("../../templates"); err != nil {
		t.Fatalf("LoadTemplates failed: %v", err)
	}
}
