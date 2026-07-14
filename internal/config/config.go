package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL          string
	JWTSecret            string
	JWTRefreshSecret     string
	EncryptionKey        string
	AllowedOrigins       []string
	AllowedInstances     []string
	NotificationInterval int
	Port                 string
	LogLevel             string
	HostResolveMap       map[string]string
	NotifyEventURL       string
}

func Load() *Config {
	loadDotEnv()

	cfg := &Config{
		DatabaseURL:          requireEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable"),
		JWTSecret:            requireEnv("JWT_SECRET", ""),
		JWTRefreshSecret:     requireEnv("JWT_REFRESH_SECRET", ""),
		EncryptionKey:        requireEnv("ENCRYPTION_KEY", ""),
		AllowedOrigins:       parseList(getEnv("ALLOWED_ORIGINS", "*")),
		AllowedInstances:     parseList(getEnv("ALLOWED_INSTANCES", "")),
		NotificationInterval: getEnvInt("NOTIFICATION_INTERVAL", 60),
		Port:                 getEnv("PORT", "8080"),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		HostResolveMap:       parseHostMap(getEnv("HOST_RESOLVE_MAP", "")),
		NotifyEventURL:       getEnv("NOTIFY_EVENT_URL", ""),
	}
	return cfg
}

func (c *Config) IsDev() bool {
	return c.LogLevel == "debug"
}

func (c *Config) IsInstanceAllowed(host string) bool {
	for _, h := range c.AllowedInstances {
		if strings.EqualFold(h, host) {
			return true
		}
	}
	return false
}

func (c *Config) ResolveHost(host string) string {
	if mapped, ok := c.HostResolveMap[host]; ok {
		return mapped
	}
	withoutScheme := host
	if strings.HasPrefix(host, "http://") {
		withoutScheme = host[7:]
	} else if strings.HasPrefix(host, "https://") {
		withoutScheme = host[8:]
	}
	if mapped, ok := c.HostResolveMap[withoutScheme]; ok {
		return mapped
	}
	withScheme := "http://" + host
	if mapped, ok := c.HostResolveMap[withScheme]; ok {
		return mapped
	}
	return host
}

func requireEnv(key, defaultForEmpty string) string {
	v := os.Getenv(key)
	if v == "" && defaultForEmpty != "" {
		return defaultForEmpty
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var result int
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		result = result*10 + int(c-'0')
	}
	return result
}

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseHostMap(s string) map[string]string {
	result := make(map[string]string)
	if s == "" {
		return result
	}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}

func loadDotEnv() {
	paths := []string{".env", ".env.dev"}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
		f.Close()
	}
}
