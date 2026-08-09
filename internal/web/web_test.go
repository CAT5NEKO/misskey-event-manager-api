package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
)

const stubIndex = `<!doctype html>
<html lang="ja">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>miSchedule</title>
    <meta property="og:title" content="miSchedule" />
    <meta property="og:description" content="miScheduleで予定を共有しよう" />
    <meta property="og:type" content="website" />
    <meta property="og:site_name" content="miSchedule" />
    <meta name="twitter:card" content="summary" />
    <script type="module" crossorigin src="/assets/index-test.js"></script>
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>`

func newTestSPA(t *testing.T, get EventTitleGetter) *SPA {
	t.Helper()
	fsys := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte(stubIndex)}}
	spa, err := NewSPA(fsys, get)
	if err != nil {
		t.Fatalf("NewSPA: %v", err)
	}
	return spa
}

func TestSPAInjectsEventOGP(t *testing.T) {
	get := func(id uuid.UUID) (string, error) { return "パーティー<&>", nil }
	spa := newTestSPA(t, get)

	req := httptest.NewRequest("GET", "/events/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	spa.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(body, "<title>パーティー&lt;&amp;&gt;</title>") {
		t.Errorf("title not injected: %s", body)
	}
	if !strings.Contains(body, `property="og:title" content="パーティー&lt;&amp;&gt;"`) {
		t.Errorf("og:title not injected: %s", body)
	}
	if !strings.Contains(body, `property="og:description" content="`+OGPDescription+`"`) {
		t.Errorf("og:description not injected: %s", body)
	}
	if !strings.Contains(body, `property="og:url" content="http://example.com/events/`) {
		t.Errorf("og:url not injected: %s", body)
	}
	if !strings.Contains(body, `src="/assets/index-test.js"`) {
		t.Errorf("base html mangled: %s", body)
	}
}

func TestSPANotFoundEventServesPlainIndex(t *testing.T) {
	get := func(id uuid.UUID) (string, error) { return "", nil }
	spa := newTestSPA(t, get)

	req := httptest.NewRequest("GET", "/events/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	spa.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `<title>パーティー`) {
		t.Errorf("event title leaked for missing event")
	}
	if !strings.Contains(rec.Body.String(), "<title>miSchedule</title>") {
		t.Errorf("expected plain index, got: %s", rec.Body.String())
	}
}

func TestSPAUnknownAPIReturns404(t *testing.T) {
	spa := newTestSPA(t, func(id uuid.UUID) (string, error) { return "x", nil })
	req := httptest.NewRequest("GET", "/api/nope", nil)
	rec := httptest.NewRecorder()
	spa.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAbsoluteURL(t *testing.T) {
	r := httptest.NewRequest("GET", "/events/abc", nil)
	r.Host = "example.com"
	if got := AbsoluteURL(r); got != "http://example.com/events/abc" {
		t.Fatalf("AbsoluteURL = %q", got)
	}
	r.Header.Set("X-Forwarded-Proto", "https")
	if got := AbsoluteURL(r); got != "https://example.com/events/abc" {
		t.Fatalf("AbsoluteURL with X-Forwarded-Proto = %q", got)
	}
	r.Header.Set("X-Forwarded-Host", "app.netlify.app")
	if got := AbsoluteURL(r); got != "https://app.netlify.app/events/abc" {
		t.Fatalf("AbsoluteURL with X-Forwarded-Host = %q", got)
	}
}

func TestEventPageCanonicalURL(t *testing.T) {
	get := func(id uuid.UUID) (string, error) { return "セミナー", nil }
	spa := newTestSPA(t, get)
	id := uuid.New()

	req := httptest.NewRequest("GET", "/public/events/"+id.String(), nil)
	req.Header.Set("X-Forwarded-Host", "app.netlify.app")
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	spa.EventPage(rec, req, id)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(body, `property="og:title" content="セミナー"`) {
		t.Errorf("og:title not injected: %s", body)
	}
	if !strings.Contains(body, `property="og:url" content="https://app.netlify.app/events/`+id.String()+`"`) {
		t.Errorf("canonical og:url missing: %s", body)
	}
}
