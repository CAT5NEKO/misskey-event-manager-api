package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"miSchedule/internal/config"
	"miSchedule/internal/model"

	"github.com/google/uuid"
)

const miAuthSessionTTL = 10 * time.Minute

type MiAuthClient struct {
	cfg          *config.Config
	mu           sync.Mutex
	sessionHosts map[string]sessionHostEntry
}

type sessionHostEntry struct {
	host      string
	expiresAt time.Time
}

func NewMiAuthClient(cfg *config.Config) *MiAuthClient {
	client := &MiAuthClient{cfg: cfg, sessionHosts: make(map[string]sessionHostEntry)}
	go client.cleanupLoop()
	return client
}

func (m *MiAuthClient) GetSessionHost(sessionID string) string {
	m.mu.Lock()
	entry, ok := m.sessionHosts[sessionID]
	delete(m.sessionHosts, sessionID)
	m.mu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return ""
	}
	return entry.host
}

func (m *MiAuthClient) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, entry := range m.sessionHosts {
			if now.After(entry.expiresAt) {
				delete(m.sessionHosts, id)
			}
		}
		m.mu.Unlock()
	}
}

func (m *MiAuthClient) resolveHost(host string) string {
	return m.cfg.ResolveHost(host)
}

func (m *MiAuthClient) CreateSession(host string) (*model.MiAuthSession, error) {
	sessionID := uuid.New().String()
	csrfToken := uuid.New().String()

	callbackURL := ""
	if origins := m.cfg.AllowedOrigins; len(origins) > 0 {
		callbackURL = origins[0] + "/auth/callback"
	}

	var hostClean string
	if len(host) >= 7 && host[:7] == "http://" {
		hostClean = host[7:]
	} else if len(host) >= 8 && host[:8] == "https://" {
		hostClean = host[8:]
	} else {
		hostClean = host
	}
	for len(hostClean) > 0 && hostClean[len(hostClean)-1] == '/' {
		hostClean = hostClean[:len(hostClean)-1]
	}

	baseURL := fmt.Sprintf("http://%s", hostClean)
	if m.cfg.IsDev() {
		baseURL = fmt.Sprintf("http://%s", hostClean)
	} else {
		baseURL = fmt.Sprintf("https://%s", hostClean)
	}
	miauthURL := fmt.Sprintf("%s/miauth/%s?name=miSchedule&permission=read:account,write:notifications",
		baseURL, sessionID)
	if callbackURL != "" {
		miauthURL += fmt.Sprintf("&callback=%s", callbackURL)
	}

	m.mu.Lock()
	m.sessionHosts[sessionID] = sessionHostEntry{
		host:      baseURL,
		expiresAt: time.Now().Add(miAuthSessionTTL),
	}
	m.mu.Unlock()

	return &model.MiAuthSession{
		ID:        sessionID,
		Host:      baseURL,
		CSRFToken: csrfToken,
		MiauthURL: miauthURL,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MiAuthClient) GetToken(session *model.MiAuthSession) (*model.MiAuthCheckResponse, error) {
	if time.Since(session.CreatedAt) > 10*time.Minute {
		return nil, fmt.Errorf("セッションの有効期限が切れています")
	}

	checkURL := fmt.Sprintf("%s/api/miauth/%s/check", m.resolveHost(session.Host), session.ID)
	req, err := http.NewRequest("POST", checkURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result model.MiAuthCheckResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
	}

	if result.Token == "" {
		return nil, fmt.Errorf("認証が完了していません。Misskey側で許可してください")
	}

	return &result, nil
}
