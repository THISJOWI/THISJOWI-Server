package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Addr        string
	K8sAuthRole string
	DBRole      string
}

type Credentials struct {
	Username      string
	Password      string
	LeaseID       string
	LeaseDuration time.Duration
	Renewable     bool
}

const (
	DefaultK8sTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	DefaultK8sAuthPath  = "kubernetes"
	DefaultDBEnginePath = "database"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

func vaultCall(ctx context.Context, addr, method, path, token string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, addr+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("vault: %s - %s", resp.Status, string(raw))
	}

	if err := json.Unmarshal(raw, result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func Login(ctx context.Context, cfg Config) (string, error) {
	jwt, err := os.ReadFile(DefaultK8sTokenPath)
	if err != nil {
		return "", fmt.Errorf("read k8s sa token at %s: %w", DefaultK8sTokenPath, err)
	}

	body := map[string]string{
		"jwt":  string(jwt),
		"role": cfg.K8sAuthRole,
	}

	var result struct {
		Auth *struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}

	if err := vaultCall(ctx, cfg.Addr, http.MethodPost, "/v1/auth/"+DefaultK8sAuthPath+"/login", "", body, &result); err != nil {
		return "", fmt.Errorf("vault k8s login: %w", err)
	}

	if result.Auth == nil || result.Auth.ClientToken == "" {
		return "", fmt.Errorf("vault login: empty client token")
	}

	return result.Auth.ClientToken, nil
}

func FetchDBCreds(ctx context.Context, clientToken string, cfg Config) (*Credentials, error) {
	var result struct {
		Data *struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"data"`
		LeaseID       string `json:"lease_id"`
		LeaseDuration int    `json:"lease_duration"`
		Renewable     bool   `json:"renewable"`
	}

	if err := vaultCall(ctx, cfg.Addr, http.MethodGet, "/v1/"+DefaultDBEnginePath+"/creds/"+cfg.DBRole, clientToken, nil, &result); err != nil {
		return nil, fmt.Errorf("vault fetch db creds: %w", err)
	}

	if result.Data == nil {
		return nil, fmt.Errorf("vault fetch db creds: empty data")
	}

	return &Credentials{
		Username:      result.Data.Username,
		Password:      result.Data.Password,
		LeaseID:       result.LeaseID,
		LeaseDuration: time.Duration(result.LeaseDuration) * time.Second,
		Renewable:     result.Renewable,
	}, nil
}

func RenewLease(ctx context.Context, clientToken, leaseID, addr string) (time.Duration, error) {
	body := map[string]string{"lease_id": leaseID}

	var result struct {
		LeaseID       string `json:"lease_id"`
		LeaseDuration int    `json:"lease_duration"`
	}

	if err := vaultCall(ctx, addr, http.MethodPut, "/v1/sys/leases/renew", clientToken, body, &result); err != nil {
		return 0, fmt.Errorf("renew lease: %w", err)
	}

	return time.Duration(result.LeaseDuration) * time.Second, nil
}

func StartRenewal(ctx context.Context, clientToken string, creds *Credentials, addr string) {
	tokenTicker := time.NewTicker(20 * time.Minute)
	defer tokenTicker.Stop()

	renewInterval := time.Duration(float64(creds.LeaseDuration) * 0.5)
	leaseTicker := time.NewTicker(renewInterval)
	defer leaseTicker.Stop()

	slog.Info("vault: starting credential renewal",
		"lease_ttl", creds.LeaseDuration,
		"renewable", creds.Renewable,
		"renew_interval", renewInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("vault: renewal loop stopped")
			return

		case <-tokenTicker.C:
			var result struct {
				Auth *struct {
					LeaseDuration int `json:"lease_duration"`
				} `json:"auth"`
			}
			if err := vaultCall(ctx, addr, http.MethodPost, "/v1/auth/token/renew-self", clientToken, nil, &result); err != nil {
				slog.Warn("vault: token renewal failed", "error", err)
				continue
			}
			if result.Auth != nil && result.Auth.LeaseDuration > 0 {
				tokenTicker.Reset(time.Duration(float64(result.Auth.LeaseDuration)*0.5) * time.Second)
			}
			slog.Debug("vault: token renewed")

		case <-leaseTicker.C:
			newDuration, err := RenewLease(ctx, clientToken, creds.LeaseID, addr)
			if err != nil {
				slog.Warn("vault: lease renewal failed", "error", err)
				continue
			}
			creds.LeaseDuration = newDuration
			leaseTicker.Reset(time.Duration(float64(newDuration) * 0.5))
			slog.Debug("vault: db lease renewed", "new_ttl", newDuration)
		}
	}
}
