package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunErrors(t *testing.T) {
	withURL := map[string]string{"DATABASE_URL": "postgres://localhost:1/x"}
	tests := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{"no command", nil, withURL},
		{"unknown command", []string{"sideways"}, withURL},
		{"missing database url", []string{"up"}, map[string]string{}},
		{"unknown flag", []string{"-bogus", "up"}, withURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(context.Background(), tt.args, func(k string) string { return tt.env[k] }, &out)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(out.String(), "usage: migrate") {
				t.Fatalf("output lacks usage: %q", out.String())
			}
		})
	}
}

func TestRunUnreachableDatabase(t *testing.T) {
	env := map[string]string{"DATABASE_URL": "postgres://counter@127.0.0.1:1/x?sslmode=disable"}
	err := run(context.Background(), []string{"up"}, func(k string) string { return env[k] }, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
}
