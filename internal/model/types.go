package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type TimeArray []time.Time

func (t *TimeArray) Scan(src interface{}) error {
	if src == nil {
		*t = nil
		return nil
	}
	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("TimeArray.Scan: expected []byte, got %T", src)
	}
	s := strings.Trim(string(str), "{}")
	if s == "" {
		*t = nil
		return nil
	}
	parts := strings.Split(s, ",")
	times := make([]time.Time, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, `" `)
		if p == "" || p == "NULL" {
			continue
		}
		parsed, err := time.Parse("2006-01-02 15:04:05", p)
		if err != nil {
			parsed, err = time.Parse("2006-01-02 15:04:05-07", p)
			if err != nil {
				parsed, err = time.Parse("2006-01-02T15:04:05Z", p)
				if err != nil {
					parsed, err = time.Parse(time.RFC3339, p)
					if err != nil {
						continue
					}
				}
			}
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
		parts[i] = fmt.Sprintf(`"%s"`, tm.Format("2006-01-02 15:04:05"))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ",")), nil
}
