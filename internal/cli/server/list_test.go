package server

import (
	"testing"
	"time"
)

func TestFormatAge_Days(t *testing.T) {
	created := time.Now().Add(-3 * 24 * time.Hour)
	result := formatAge(created)
	if result != "3d" {
		t.Errorf("formatAge() = %q, want %q", result, "3d")
	}
}

func TestFormatAge_Hours(t *testing.T) {
	created := time.Now().Add(-5 * time.Hour)
	result := formatAge(created)
	if result != "5h" {
		t.Errorf("formatAge() = %q, want %q", result, "5h")
	}
}

func TestFormatAge_Minutes(t *testing.T) {
	created := time.Now().Add(-30 * time.Minute)
	result := formatAge(created)
	if result != "30m" {
		t.Errorf("formatAge() = %q, want %q", result, "30m")
	}
}

func TestFormatAge_LessThanOneMinute(t *testing.T) {
	created := time.Now().Add(-10 * time.Second)
	result := formatAge(created)
	if result != "0m" {
		t.Errorf("formatAge() = %q, want %q", result, "0m")
	}
}

func TestFormatAge_ExactlyOneDay(t *testing.T) {
	created := time.Now().Add(-24 * time.Hour)
	result := formatAge(created)
	if result != "1d" {
		t.Errorf("formatAge() = %q, want %q", result, "1d")
	}
}

func TestFormatAge_ExactlyOneHour(t *testing.T) {
	created := time.Now().Add(-1 * time.Hour)
	result := formatAge(created)
	if result != "1h" {
		t.Errorf("formatAge() = %q, want %q", result, "1h")
	}
}
