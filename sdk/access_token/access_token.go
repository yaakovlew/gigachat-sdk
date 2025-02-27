package accesstoken

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/yaakovlew/gigachat_sdk/certificates"
	"github.com/yaakovlew/gigachat_sdk/config"
)

type GigaChatAccessToken struct {
	mu sync.RWMutex

	model       string
	url         string
	baseToken   string
	clientToken string
	scopeValue  string

	gigaToken jwtToken

	cert certificates.Certificate
}

type jwtToken struct {
	accessToken string
	expiresAt   int64
}

func NewGigaChatToken(cfg config.GigaChatConfig, cert certificates.Certificate) *GigaChatAccessToken {
	token := &GigaChatAccessToken{
		model:       cfg.Model,
		baseToken:   cfg.BaseToken,
		url:         cfg.AuthUrl,
		clientToken: cfg.ClientToken,
		scopeValue:  cfg.Scope,

		cert: cert,
	}

	if err := token.updateJWT(); err != nil {
		log.Fatal(err)
	}

	go token.refresh()

	return token
}

func (token *GigaChatAccessToken) updateJWT() error {
	// For phys face
	data := url.Values{}
	data.Set("scope", token.scopeValue)

	req, err := http.NewRequest("POST", token.url, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("RqUID", token.clientToken)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", token.baseToken))

	resp, err := token.cert.HttpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update jwt failed, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		return fmt.Errorf("update jwt failed, access token is not a string")
	}
	expiresAt, ok := result["expires_at"].(float64)
	if !ok {
		return fmt.Errorf("update jwt failed, expires_at is not a float64")
	}

	token.gigaToken.accessToken = accessToken
	token.gigaToken.expiresAt = int64(expiresAt)

	return err
}

func (token *GigaChatAccessToken) JwtToken() string {
	token.mu.RLock()
	defer token.mu.RUnlock()

	return token.gigaToken.accessToken
}

func (token *GigaChatAccessToken) ExpiresJWTTime() int64 {
	token.mu.RLock()
	defer token.mu.RUnlock()

	return token.gigaToken.expiresAt
}

func (token *GigaChatAccessToken) refresh() {
	ticker := time.NewTicker(time.Millisecond)

	for range ticker.C {
		if err := token.updateJWT(); err != nil {
			log.Error(err)
			continue
		}

		// 30 sec to update token
		// operation /1000 used because expiresAt in milliseconds
		if token.gigaToken.expiresAt == 0 {
			ticker.Reset(time.Second * 60)
			continue
		}

		ticker.Reset(time.Second * time.Duration(time.Unix((token.gigaToken.expiresAt/1000-30), 0).Sub(time.Now()).Seconds()))
	}
}

func (token *GigaChatAccessToken) Model() string {
	token.mu.RLock()
	defer token.mu.RUnlock()

	return token.model
}
