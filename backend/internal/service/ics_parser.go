package service

import (
	"strings"
	"time"
)

// ICSEvent represents a parsed VEVENT from an iCalendar file.
type ICSEvent struct {
	Start       time.Time
	End         time.Time
	Summary     string
	Description string
}

// ParseICS parses an iCalendar (.ics) string and returns all VEVENT entries.
func ParseICS(content string, loc *time.Location) ([]ICSEvent, error) {
	var events []ICSEvent
	var current *ICSEvent
	inVevent := false

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)

		if line == "BEGIN:VEVENT" {
			inVevent = true
			current = &ICSEvent{}
			continue
		}
		if line == "END:VEVENT" {
			if current != nil && !current.Start.IsZero() && !current.End.IsZero() {
				events = append(events, *current)
			}
			inVevent = false
			current = nil
			continue
		}
		if !inVevent || current == nil {
			continue
		}

		if strings.HasPrefix(line, "DTSTART:") || strings.HasPrefix(line, "DTSTART;") {
			val := extractICSValue(line)
			if t, err := parseICSTime(val, loc); err == nil {
				current.Start = t
			}
		} else if strings.HasPrefix(line, "DTEND:") || strings.HasPrefix(line, "DTEND;") {
			val := extractICSValue(line)
			if t, err := parseICSTime(val, loc); err == nil {
				current.End = t
			}
		} else if strings.HasPrefix(line, "SUMMARY:") {
			current.Summary = unescapeICS(line[len("SUMMARY:"):])
		} else if strings.HasPrefix(line, "DESCRIPTION:") {
			current.Description = unescapeICS(line[len("DESCRIPTION:"):])
		}
	}

	return events, nil
}

func extractICSValue(line string) string {
	if idx := strings.IndexByte(line, ':'); idx != -1 {
		return line[idx+1:]
	}
	return ""
}

func parseICSTime(val string, loc *time.Location) (time.Time, error) {
	val = strings.TrimSpace(val)
	if len(val) >= 15 {
		return time.ParseInLocation("20060102T150405", val[:15], loc)
	}
	return time.Time{}, nil
}

func unescapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\,", ",")
	s = strings.ReplaceAll(s, "\\;", ";")
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}
