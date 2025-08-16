package main

import (
	"os"
	"testing"
)

func TestPrintEnvMasked(t *testing.T) {
	os.Setenv("TEST_KEY1", "value1")
	os.Setenv("TEST_KEY2", "")
	// This just checks that the function runs without error
	printEnvMasked([]string{"TEST_KEY1", "TEST_KEY2", "NOT_SET"})
}
