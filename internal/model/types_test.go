package model

import (
	"testing"
	"time"
)

func TestTimeArrayValueMarksUnsetEntriesAsNull(t *testing.T) {
	arr := TimeArray{
		time.Time{},
		time.Date(2026, 11, 27, 0, 8, 51, 86257000, time.FixedZone("JST", 9*3600)),
	}

	got, err := arr.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	want := `{NULL,"2026-11-27T00:08:51.086257+09:00"}`
	if got != want {
		t.Fatalf("Value = %v, want %v", got, want)
	}
}

func TestTimeArrayScanPreservesUnsetEntries(t *testing.T) {
	cases := []string{
		`{"0001-01-01 00:18:59+09:18:59","2026-11-27 00:08:55.600006+09"}`,
		`{NULL,"2026-11-27 00:08:55.600006+09"}`,
	}

	for _, src := range cases {
		var arr TimeArray
		if err := arr.Scan([]byte(src)); err != nil {
			t.Fatalf("Scan(%s): %v", src, err)
		}
		if len(arr) != 2 {
			t.Fatalf("Scan(%s): len = %d, want 2", src, len(arr))
		}
		if !arr[0].IsZero() {
			t.Errorf("Scan(%s): arr[0] = %s, want zero", src, arr[0])
		}
		if arr[1].IsZero() {
			t.Errorf("Scan(%s): arr[1] must not be zero", src)
		}
	}
}

func TestTimeArrayRoundTrip(t *testing.T) {
	now := time.Date(2026, 11, 27, 0, 8, 51, 86257000, time.FixedZone("JST", 9*3600))
	arr := TimeArray{time.Time{}, now}

	value, err := arr.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}

	var got TimeArray
	if err := got.Scan(value); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != len(arr) {
		t.Fatalf("len = %d, want %d", len(got), len(arr))
	}
	if !got[0].IsZero() {
		t.Errorf("got[0] = %s, want zero", got[0])
	}
	if !got[1].Equal(now) {
		t.Errorf("got[1] = %s, want %s", got[1], now)
	}
}
