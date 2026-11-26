package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

var timeLayouts = []string{
	"2006-01-02 15:04:05.999999999Z07:00:00",
	"2006-01-02 15:04:05.999999999-07:00:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05Z07:00:00",
	"2006-01-02 15:04:05-07:00:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05-07",
	"2006-01-02 15:04:05",
	time.RFC3339Nano,
	time.RFC3339,
}

type TimeArray []time.Time

func parseTimeArrayValue(value string) (time.Time, bool) {
	for _, layout := range timeLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func (t *TimeArray) Scan(src interface{}) error {
	if src == nil {
		*t = nil
		return nil
	}

	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("TimeArray.Scan: expected []byte or string, got %T", src)
	}

	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	if strings.TrimSpace(s) == "" {
		*t = nil
		return nil
	}

	parts := strings.Split(s, ",")
	times := make([]time.Time, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || strings.EqualFold(p, "NULL") {
			times = append(times, time.Time{})
			continue
		}

		p = strings.Trim(p, `"`)
		parsed, ok := parseTimeArrayValue(p)
		if !ok || parsed.Year() <= 1 {
			times = append(times, time.Time{})
			continue
		}
		times = append(times, parsed)
	}
	*t = TimeArray(times)
	return nil
}

func (t TimeArray) Value() (driver.Value, error) {
	if t == nil {
		return "{}", nil
	}
	parts := make([]string, len(t))
	for i, tm := range t {
		if tm.IsZero() {
			parts[i] = "NULL"
			continue
		}
		parts[i] = fmt.Sprintf(`"%s"`, tm.Format(time.RFC3339Nano))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ",")), nil
}
