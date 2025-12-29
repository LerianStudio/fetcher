package pkg

import (
	"testing"
	"time"
)

func TestIsValidDate(t *testing.T) {
	tests := []struct {
		name string
		date string
		want bool
	}{
		{
			name: "valid date",
			date: "2025-01-15",
			want: true,
		},
		{
			name: "valid leap year date",
			date: "2024-02-29",
			want: true,
		},
		{
			name: "invalid format - wrong separator",
			date: "2025/01/15",
			want: false,
		},
		{
			name: "invalid format - wrong order",
			date: "15-01-2025",
			want: false,
		},
		{
			name: "invalid date - Feb 30",
			date: "2025-02-30",
			want: false,
		},
		{
			name: "invalid date - Feb 29 non-leap year",
			date: "2025-02-29",
			want: false,
		},
		{
			name: "invalid date - month 13",
			date: "2025-13-01",
			want: false,
		},
		{
			name: "empty string",
			date: "",
			want: false,
		},
		{
			name: "random string",
			date: "not-a-date",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidDate(tt.date); got != tt.want {
				t.Errorf("IsValidDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInitialDateBeforeFinalDate(t *testing.T) {
	baseTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		initial time.Time
		final   time.Time
		want    bool
	}{
		{
			name:    "initial before final",
			initial: baseTime,
			final:   baseTime.AddDate(0, 0, 10),
			want:    true,
		},
		{
			name:    "initial equals final",
			initial: baseTime,
			final:   baseTime,
			want:    true,
		},
		{
			name:    "initial after final",
			initial: baseTime.AddDate(0, 0, 10),
			final:   baseTime,
			want:    false,
		},
		{
			name:    "initial one day before final",
			initial: baseTime,
			final:   baseTime.AddDate(0, 0, 1),
			want:    true,
		},
		{
			name:    "initial one second before final",
			initial: baseTime,
			final:   baseTime.Add(time.Second),
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInitialDateBeforeFinalDate(tt.initial, tt.final); got != tt.want {
				t.Errorf("IsInitialDateBeforeFinalDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDateRangeWithinMonthLimit(t *testing.T) {
	baseTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		initial time.Time
		final   time.Time
		limit   int
		want    bool
	}{
		{
			name:    "within 1 month limit",
			initial: baseTime,
			final:   baseTime.AddDate(0, 0, 15),
			limit:   1,
			want:    true,
		},
		{
			name:    "exactly at 1 month limit",
			initial: baseTime,
			final:   baseTime.AddDate(0, 1, 0),
			limit:   1,
			want:    true,
		},
		{
			name:    "exceeds 1 month limit",
			initial: baseTime,
			final:   baseTime.AddDate(0, 1, 1),
			limit:   1,
			want:    false,
		},
		{
			name:    "within 3 month limit",
			initial: baseTime,
			final:   baseTime.AddDate(0, 2, 0),
			limit:   3,
			want:    true,
		},
		{
			name:    "exceeds 3 month limit",
			initial: baseTime,
			final:   baseTime.AddDate(0, 4, 0),
			limit:   3,
			want:    false,
		},
		{
			name:    "same day",
			initial: baseTime,
			final:   baseTime,
			limit:   1,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDateRangeWithinMonthLimit(tt.initial, tt.final, tt.limit); got != tt.want {
				t.Errorf("IsDateRangeWithinMonthLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeDate(t *testing.T) {
	baseTime := time.Date(2025, 6, 15, 12, 30, 45, 0, time.UTC)
	days1 := 1
	days5 := 5
	daysNeg3 := -3

	tests := []struct {
		name string
		date time.Time
		days *int
		want string
	}{
		{
			name: "no days adjustment",
			date: baseTime,
			days: nil,
			want: "2025-06-15",
		},
		{
			name: "add 1 day",
			date: baseTime,
			days: &days1,
			want: "2025-06-16",
		},
		{
			name: "add 5 days",
			date: baseTime,
			days: &days5,
			want: "2025-06-20",
		},
		{
			name: "subtract 3 days",
			date: baseTime,
			days: &daysNeg3,
			want: "2025-06-12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeDate(tt.date, tt.days); got != tt.want {
				t.Errorf("NormalizeDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
