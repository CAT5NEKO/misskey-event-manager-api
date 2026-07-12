package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"miSchedule/internal/config"
	"miSchedule/internal/model"

	"github.com/google/uuid"
)

type MiAuthClient struct {
	cfg          *config.Config
	sessionHosts map[string]string
}

func NewMiAuthClient(cfg *config.Config) *MiAuthClient {
	return &MiAuthClient{cfg: cfg, sessionHosts: make(map[string]string)}
}

func (m *MiAuthClient) GetSessionHost(sessionID string) string {
	host := m.sessionHosts[sessionID]
	delete(m.sessionHosts, sessionID)
	return host
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

	m.sessionHosts[sessionID] = baseURL

	return &model.MiAuthSession{
		ID:        sessionID,
		Host:      baseURL,
		CSRFToken: csrfToken,
		MiauthURL:  miauthURL,
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

func (m *MiAuthClient) RevokeToken(host, token string) error {
	host = m.resolveHost(host)
	apiURL := fmt.Sprintf("%s/api/i/revoke-token", host)
	reqBody := map[string]string{}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		return fmt.Errorf("トークン失効に失敗しました (status %d)", resp.StatusCode)
	}
	return nil
}
