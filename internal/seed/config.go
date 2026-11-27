// Package seed generates deterministic synthetic retail data for a fictional
// jewelry chain and writes it to Postgres. It is used by cmd/seed and tests
// only; the service must not import it.
package seed

import (
	"errors"
	"time"

	_ "time/tzdata" // branch zones must not depend on the OS zone database
)

const (
	maxBranches           = 15
	maxEmployeesPerBranch = 99
)

// Config controls what Generate produces. The same Config always produces
// the same Dataset.
type Config struct {
	Seed               uint64
	Start              time.Time // first day of the window; only the date is used
	Months             int
	Timezone           string
	Branches           int
	Items              int
	EmployeesPerBranch int
	MinSalesPerDay     int // per branch
	MaxSalesPerDay     int // per branch
}

// Default returns the full-size configuration: 15 branches and six whole
// months starting 2026-05-01.
func Default() Config {
	return Config{
		Seed:               1,
		Start:              time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Months:             6,
		Timezone:           "America/Mexico_City",
		Branches:           maxBranches,
		Items:              300,
		EmployeesPerBranch: 4,
		MinSalesPerDay:     15,
		MaxSalesPerDay:     45,
	}
}

// Validate reports every invalid field.
func (c Config) Validate() error {
	var errs []error
	if c.Start.IsZero() {
		errs = append(errs, errors.New("start date is required"))
	}
	if c.Months < 1 {
		errs = append(errs, errors.New("months must be at least 1"))
	}
	if c.Branches < 1 || c.Branches > maxBranches {
		errs = append(errs, errors.New("branches must be between 1 and 15"))
	}
	if c.Items < 1 {
		errs = append(errs, errors.New("items must be at least 1"))
	}
	if c.EmployeesPerBranch < 1 || c.EmployeesPerBranch > maxEmployeesPerBranch {
		errs = append(errs, errors.New("employees per branch must be between 1 and 99"))
	}
	if c.MinSalesPerDay < 0 || c.MaxSalesPerDay < c.MinSalesPerDay || c.MaxSalesPerDay < 1 {
		errs = append(errs, errors.New("sales per day must satisfy 0 <= min <= max and max >= 1"))
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		errs = append(errs, errors.New("unknown timezone "+c.Timezone))
	}
	return errors.Join(errs...)
}
