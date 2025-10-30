package daemon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCronExprParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *CronExpr
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid parse - 14 30",
			input:   "14 30",
			want:    &CronExpr{Hour: 14, Min: 30},
			wantErr: false,
		},
		{
			name:    "valid parse - 0 0",
			input:   "0 0",
			want:    &CronExpr{Hour: 0, Min: 0},
			wantErr: false,
		},
		{
			name:    "valid parse - 23 59",
			input:   "23 59",
			want:    &CronExpr{Hour: 23, Min: 59},
			wantErr: false,
		},
		{
			name:    "valid parse with extra spaces",
			input:   "  12  45  ",
			want:    &CronExpr{Hour: 12, Min: 45},
			wantErr: false,
		},
		{
			name:    "invalid - missing minute",
			input:   "14",
			wantErr: true,
			errMsg:  "invalid schedule format",
		},
		{
			name:    "invalid - too many parts",
			input:   "14 30 00",
			wantErr: true,
			errMsg:  "invalid schedule format",
		},
		{
			name:    "invalid - non-numeric hour",
			input:   "abc 30",
			wantErr: true,
			errMsg:  "invalid hour",
		},
		{
			name:    "invalid - non-numeric minute",
			input:   "14 xyz",
			wantErr: true,
			errMsg:  "invalid minute",
		},
		{
			name:    "invalid - hour out of range (negative)",
			input:   "-1 30",
			wantErr: true,
			errMsg:  "invalid hour",
		},
		{
			name:    "invalid - hour out of range (too large)",
			input:   "24 30",
			wantErr: true,
			errMsg:  "invalid hour",
		},
		{
			name:    "invalid - minute out of range (negative)",
			input:   "14 -1",
			wantErr: true,
			errMsg:  "invalid minute",
		},
		{
			name:    "invalid - minute out of range (too large)",
			input:   "14 60",
			wantErr: true,
			errMsg:  "invalid minute",
		},
		{
			name:    "invalid - empty string",
			input:   "",
			wantErr: true,
			errMsg:  "invalid schedule format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := &CronExpr{}
			err := ce.Parse(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want.Hour, ce.Hour)
				require.Equal(t, tt.want.Min, ce.Min)
			}
		})
	}
}

func TestCronExprNext(t *testing.T) {
	utcLoc := time.UTC
	csaLoc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	tests := []struct {
		name     string
		cronHour int
		cronMin  int
		now      time.Time
		want     time.Time
	}{
		{
			name:     "next trigger is today (future time) - UTC",
			cronHour: 15,
			cronMin:  30,
			now:      time.Date(2025, 10, 30, 14, 0, 0, 0, utcLoc),
			want:     time.Date(2025, 10, 30, 15, 30, 0, 0, utcLoc),
		},
		{
			name:     "next trigger is tomorrow (past time today) - UTC",
			cronHour: 10,
			cronMin:  0,
			now:      time.Date(2025, 10, 30, 14, 0, 0, 0, utcLoc),
			want:     time.Date(2025, 10, 31, 10, 0, 0, 0, utcLoc),
		},
		{
			name:     "next trigger is tomorrow (exact same time) - UTC",
			cronHour: 14,
			cronMin:  0,
			now:      time.Date(2025, 10, 30, 14, 0, 0, 0, utcLoc),
			want:     time.Date(2025, 10, 31, 14, 0, 0, 0, utcLoc),
		},
		{
			name:     "midnight trigger from morning - UTC",
			cronHour: 0,
			cronMin:  0,
			now:      time.Date(2025, 10, 30, 8, 30, 0, 0, utcLoc),
			want:     time.Date(2025, 10, 31, 0, 0, 0, 0, utcLoc),
		},
		{
			name:     "end of day trigger from early morning - UTC",
			cronHour: 23,
			cronMin:  59,
			now:      time.Date(2025, 10, 30, 1, 0, 0, 0, utcLoc),
			want:     time.Date(2025, 10, 30, 23, 59, 0, 0, utcLoc),
		},
		{
			name:     "end of day trigger from late evening - UTC",
			cronHour: 23,
			cronMin:  59,
			now:      time.Date(2025, 10, 30, 23, 59, 1, 0, utcLoc),
			want:     time.Date(2025, 10, 31, 23, 59, 0, 0, utcLoc),
		},
		{
			name:     "first second of day - UTC",
			cronHour: 0,
			cronMin:  0,
			now:      time.Date(2025, 10, 30, 0, 0, 0, 0, utcLoc),
			want:     time.Date(2025, 10, 31, 0, 0, 0, 0, utcLoc),
		},
		{
			name:     "preserves timezone - Asia/Shanghai",
			cronHour: 10,
			cronMin:  30,
			now:      time.Date(2025, 10, 30, 8, 0, 0, 0, csaLoc),
			want:     time.Date(2025, 10, 30, 10, 30, 0, 0, csaLoc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := &CronExpr{Hour: tt.cronHour, Min: tt.cronMin}
			next := ce.Next(tt.now)

			require.Equal(t, tt.want, next)
			require.True(t, next.After(tt.now), "Next() should return time after the given time")
		})
	}
}
