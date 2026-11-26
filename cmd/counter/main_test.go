package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const unreachableDB = "postgres://counter:counter@127.0.0.1:1/counter?sslmode=disable"

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"missing config", map[string]string{}},
		{"unreachable database", map[string]string{"ADDR": "127.0.0.1:0", "DATABASE_URL": unreachableDB}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(context.Background(), func(k string) string { return tt.env[k] }, io.Discard)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRunShutsDownOnCancel(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	env := map[string]string{"ADDR": "127.0.0.1:0", "DATABASE_URL": url}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, func(k string) string { return env[k] }, io.Discard)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after cancel")
	}
}

func TestRunBadAddress(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	env := map[string]string{"ADDR": "bad:addr:x", "DATABASE_URL": url}
	if err := run(context.Background(), func(k string) string { return env[k] }, io.Discard); err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceDoesNotImportMigrations(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, banned := range []string{"github.com/pressly/goose", "github.com/su1uv/counter/internal/migrate"} {
		if strings.Contains(string(out), banned) {
			t.Errorf("service imports %s", banned)
		}
	}
}
