package service

import (
	"fmt"
	"time"
)

// ValidationMismatch holds info about a single preference mismatch.
type ValidationMismatch struct {
	TaskID         string
	TaskTitle      string
	PreferredStart string
	PreferredEnd   string
	ActualStart    time.Time
	ActualEnd      time.Time
}

// RecurrenceMismatch holds info about a recurrence day mismatch.
type RecurrenceMismatch struct {
	TaskID    string
	TaskTitle string
	Date      string
	Reason    string
	Expected  string
	Actual    string
}

func (m RecurrenceMismatch) String() string {
	if m.Date != "" {
		return fmt.Sprintf("%s %s: %s (期望 %s, 实际 %s)", m.Date, m.TaskTitle, m.Reason, m.Expected, m.Actual)
	}
	return fmt.Sprintf("%s: %s (期望 %s, 实际 %s)", m.TaskTitle, m.Reason, m.Expected, m.Actual)
}
