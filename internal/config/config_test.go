package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{"all set", map[string]string{"ADDR": ":9000", "DATABASE_URL": "postgres://x"}, Config{Addr: ":9000", DatabaseURL: "postgres://x"}, false},
		{"addr empty uses default", map[string]string{"DATABASE_URL": "postgres://x"}, Config{Addr: ":8080", DatabaseURL: "postgres://x"}, false},
		{"database url missing", map[string]string{"ADDR": ":9000"}, Config{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(func(k string) string { return tt.env[k] })
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
