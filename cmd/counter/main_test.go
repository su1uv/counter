package main

import (
	"context"
	"io"
	"testing"
	"time"
)

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"missing config", map[string]string{}},
		{"bad address", map[string]string{"ADDR": "bad:addr:x", "DATABASE_URL": "x"}},
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
	env := map[string]string{"ADDR": "127.0.0.1:0", "DATABASE_URL": "x"}
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
