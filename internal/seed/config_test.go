package seed

import (
	"strings"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Default().Validate() = %v", err)
	}
	if want := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC); !cfg.Start.Equal(want) {
		t.Fatalf("Start = %v, want %v", cfg.Start, want)
	}
	if cfg.Months != 6 || cfg.Branches != 15 || cfg.Seed != 1 {
		t.Fatalf("Default() = %+v", cfg)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(*Config)
		wantErr string
	}{
		{"default is valid", func(*Config) {}, ""},
		{"zero start", func(c *Config) { c.Start = time.Time{} }, "start"},
		{"zero months", func(c *Config) { c.Months = 0 }, "months"},
		{"zero branches", func(c *Config) { c.Branches = 0 }, "branches"},
		{"too many branches", func(c *Config) { c.Branches = maxBranches + 1 }, "branches"},
		{"zero items", func(c *Config) { c.Items = 0 }, "items"},
		{"zero employees", func(c *Config) { c.EmployeesPerBranch = 0 }, "employees"},
		{"too many employees", func(c *Config) { c.EmployeesPerBranch = 100 }, "employees"},
		{"negative min sales", func(c *Config) { c.MinSalesPerDay = -1 }, "sales per day"},
		{"max below min", func(c *Config) { c.MinSalesPerDay, c.MaxSalesPerDay = 5, 4 }, "sales per day"},
		{"zero max sales", func(c *Config) { c.MinSalesPerDay, c.MaxSalesPerDay = 0, 0 }, "sales per day"},
		{"unknown timezone", func(c *Config) { c.Timezone = "Mars/Olympus" }, "timezone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.edit(&cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
