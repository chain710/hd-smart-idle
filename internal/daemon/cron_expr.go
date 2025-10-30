package daemon

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronExpr parses and represents a simple cron expression.
// For now only supports "hour min" format for daily scheduling.
type CronExpr struct {
	Hour int
	Min  int
}

// Parse 从 "hour min" 格式的字符串解析时间表
// 例如: "14 30" 表示每天 14:30 触发
func (ce *CronExpr) Parse(expr string) error {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 2 {
		return fmt.Errorf("invalid schedule format: expected 'hour min', got %q", expr)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid hour: %w", err)
	}

	min, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid minute: %w", err)
	}

	if hour < 0 || hour > 23 {
		return fmt.Errorf("invalid hour: %d (must be 0-23)", hour)
	}

	if min < 0 || min > 59 {
		return fmt.Errorf("invalid minute: %d (must be 0-59)", min)
	}

	ce.Hour = hour
	ce.Min = min
	return nil
}

// Next 返回从给定时间开始，下一次触发的时间
func (ce *CronExpr) Next(t time.Time) time.Time {
	loc := t.Location()
	next := time.Date(t.Year(), t.Month(), t.Day(), ce.Hour, ce.Min, 0, 0, loc)

	// 如果下一次触发时间已经过了，加一天
	if !next.After(t) {
		next = next.Add(24 * time.Hour)
	}

	return next
}

// String implements flag.Value interface
func (ce *CronExpr) String() string {
	return fmt.Sprintf("%d %d", ce.Hour, ce.Min)
}

// Set implements flag.Value interface
func (ce *CronExpr) Set(value string) error {
	return ce.Parse(value)
}

// Type implements pflag.Value interface (optional, for better help text)
func (ce *CronExpr) Type() string {
	return "hour min"
}
