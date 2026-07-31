package main

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestGetenv(t *testing.T) {
	os.Setenv("TEST_KEY_GO_CINEMA", "val123")
	defer os.Unsetenv("TEST_KEY_GO_CINEMA")

	if got := getenv("TEST_KEY_GO_CINEMA", "default"); got != "val123" {
		t.Errorf("expected val123, got %s", got)
	}

	if got := getenv("NON_EXISTENT_KEY_XYZ", "default"); got != "default" {
		t.Errorf("expected default, got %s", got)
	}
}

func TestGetenvDuration(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	os.Setenv("TEST_DUR_KEY", "15m")
	defer os.Unsetenv("TEST_DUR_KEY")

	dur := getenvDuration(logger, "TEST_DUR_KEY", "1m")
	if dur != 15*time.Minute {
		t.Errorf("expected 15m, got %v", dur)
	}
}
