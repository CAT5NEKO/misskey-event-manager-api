package notifier

import (
	"testing"
	"time"

	"miSchedule/internal/model"

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

func TestNextNotifyDelay(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	interval := 60 * time.Second

	mkEvent := func(deadline time.Time, timings []int, notified []time.Time) model.Event {
		return model.Event{
			ID:                 uuid.New(),
			Deadline:           &deadline,
			NotificationTiming: timings,
			NotifiedAt:         model.TimeArray(notified),
		}
	}

	t.Run("wakes precisely when due is inside interval", func(t *testing.T) {
		// deadline = now+90s, timing 1min -> notifyAt = now+30s
		events := []model.Event{mkEvent(now.Add(90*time.Second), []int{1}, nil)}
		want := 30 * time.Second
		if got := nextNotifyDelay(events, interval, now); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("ignores past and far-future timings, caps at interval", func(t *testing.T) {
		events := []model.Event{
			mkEvent(now.Add(24*time.Hour), []int{1440, 180}, nil), // notifyAt = now / now+21h
		}
		if got := nextNotifyDelay(events, interval, now); got != interval {
			t.Errorf("got %v, want fallback interval %v", got, interval)
		}
	})

	t.Run("takes the nearest due among multiple events", func(t *testing.T) {
		events := []model.Event{
			mkEvent(now.Add(24*time.Hour), []int{180}, nil), // notifyAt = now+21h
			mkEvent(now.Add(3*time.Minute), []int{2}, nil),  // notifyAt = now+60s
			mkEvent(now.Add(4*time.Minute), []int{3}, nil),  // notifyAt = now+60s
		}
		if got := nextNotifyDelay(events, interval, now); got != interval {
			t.Errorf("got %v, want cap interval %v (due equals interval)", got, interval)
		}
	})

	t.Run("skips already notified timings", func(t *testing.T) {
		deadline := now.Add(10 * time.Minute)
		events := []model.Event{
			mkEvent(deadline, []int{1440, 1}, []time.Time{now, now}),
		}
		if got := nextNotifyDelay(events, interval, now); got != interval {
			t.Errorf("got %v, want fallback interval %v", got, interval)
		}
	})

	t.Run("no events falls back to interval", func(t *testing.T) {
		if got := nextNotifyDelay(nil, interval, now); got != interval {
			t.Errorf("got %v, want %v", got, interval)
		}
	})

	t.Run("precise wake never below one second", func(t *testing.T) {
		deadline := now.Add(90 * time.Second)
		events := []model.Event{mkEvent(deadline, []int{1}, nil)} // notifyAt = now+30s
		if got := nextNotifyDelay(events, interval, now); got < time.Second || got > 30*time.Second {
			t.Errorf("got %v, want in [1s, 30s]", got)
		}
	})

	t.Run("already-due timing falls back to interval", func(t *testing.T) {
		deadline := now.Add(30 * time.Second)
		events := []model.Event{mkEvent(deadline, []int{1}, nil)} // notifyAt = now-30s
		if got := nextNotifyDelay(events, interval, now); got != interval {
			t.Errorf("got %v, want fallback interval %v", got, interval)
		}
	})
}
