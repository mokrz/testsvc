package main

import (
	"testing"
)

func TestTestMe(t *testing.T) {
	if TestMe(4)/2 != 4 {
		t.Error("Expected 4")
	}
}
