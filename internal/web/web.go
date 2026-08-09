package web

import (
	"html"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const SiteName = "miSchedule"

const OGPDescription = "Misskeyユーザーと、スケジュールを共有しよう"

var eventIDRe = regexp.MustCompile(`^/events/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`)

var (
	metaTitleRe = regexp.MustCompile(`<title>.*?</title>`)
	ogTitleRe   = regexp.MustCompile(`(<meta property="og:title" content=")[^"]*(")`)
	ogDescRe    = regexp.MustCompile(`(<meta property="og:description" content=")[^"]*(")`)
)

// AbsoluteURL builds an absolute URL useful for og:url. Returns "" when the
// request Host is unavailable. It honors X-Forwarded-Proto / X-Forwarded-Host
// so that a reverse proxy or a CDN edge (e.g. Netlify) can force the public
// origin.
func AbsoluteURL(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
		if host == "" {
			return ""
		}
	}
	scheme := "https"
	if v := r.Header.Get("X-Forwarded-Proto"); v == "http" {
		scheme = "http"
	} else if r.TLS == nil && v == "" {
		scheme = "http"
	}
	return scheme + "://" + host + r.URL.Path
}

type EventTitleGetter func(id uuid.UUID) (string, error)

type SPA struct {
	assets   fs.FS
	base     []byte
	getTitle EventTitleGetter
}

func NewSPA(assets fs.FS, getTitle EventTitleGetter) (*SPA, error) {
	base, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return nil, err
	}
	return &SPA{assets: assets, base: base, getTitle: getTitle}, nil
}

func (s *SPA) AssetsHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})
}

func (s *SPA) EventPage(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	title, err := s.getTitle(id)
	if err != nil || title == "" {
		s.ServeHTTP(w, r)
		return
	}
	clone := r.Clone(r.Context())
	clone.URL.Path = "/events/" + id.String()
	s.writeInjected(w, clone, title)
}

func (s *SPA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(s.base) == 0 {
		http.Error(w, "frontend not built", http.StatusServiceUnavailable)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if m := eventIDRe.FindStringSubmatch(r.URL.Path); m != nil && s.getTitle != nil {
		id, err := uuid.Parse(m[1])
		if err == nil {
			if title, err := s.getTitle(id); err == nil && title != "" {
				s.writeInjected(w, r, title)
				return
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(s.base)
}

func (s *SPA) writeInjected(w http.ResponseWriter, r *http.Request, title string) {
	escaped := html.EscapeString(title)
	out := string(s.base)
	out = metaTitleRe.ReplaceAllString(out, "<title>"+escaped+"</title>")
	out = ogTitleRe.ReplaceAllString(out, "${1}"+escaped+"${2}")
	out = ogDescRe.ReplaceAllString(out, "${1}"+OGPDescription+"${2}")
	if u := AbsoluteURL(r); u != "" {
		ogURL := `<meta property="og:url" content="` + html.EscapeString(u) + `">`
		out = strings.Replace(out, "</head>", "\n  "+ogURL+"\n</head>", 1)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(out))
}
