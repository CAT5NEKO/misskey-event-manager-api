package notifier

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuildNotificationBodyTiming(t *testing.T) {
	id := uuid.MustParse("10000000-0000-4000-8000-000000000000")
	cases := []struct {
		timingMin int
		want      string
	}{
		{1440, "あと1日"},
		{2880, "あと2日"},
		{1439, "あと23時間"},
		{180, "あと3時間"},
		{60, "あと1時間"},
		{45, "あと45分"},
		{1, "あと1分"},
	}
	for _, c := range cases {
		got := buildNotificationBody("テスト", id, c.timingMin, "")
		want := "[期限] テスト（" + c.want + "）"
		if got != want {
			t.Errorf("timingMin=%d: got %q, want %q", c.timingMin, got, want)
		}
	}
}

func TestBuildNotificationBodyWithURL(t *testing.T) {
	id := uuid.MustParse("20000000-0000-4000-8000-000000000000")
	got := buildNotificationBody("テストスケジュール", id, 1440, "https://example.com")
	want := "[期限] テストスケジュール（あと1日）\nhttps://example.com/events/20000000-0000-4000-8000-000000000000"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildNotificationBodyWithoutURL(t *testing.T) {
	id := uuid.MustParse("30000000-0000-4000-8000-000000000000")
	got := buildNotificationBody("テスト", id, 180, "")
	if got != "[期限] テスト（あと3時間）" {
		t.Errorf("expected single line without trailing newline, got %q", got)
	}
	if len(got) > 0 && got[len(got)-1] == '\n' {
		t.Errorf("body must not end with newline, got %q", got)
	}
}
