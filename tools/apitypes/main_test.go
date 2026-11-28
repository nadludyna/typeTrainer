package main

import (
	"os"
	"testing"
)

// The page's API types are the Go types as they are now.
func TestUpToDate(t *testing.T) {
	have, err := os.ReadFile("../../" + out)
	if err != nil {
		t.Fatal(err)
	}
	if string(have) != generate() {
		t.Fatal(out + " is out of date: go run ./tools/apitypes")
	}
}
