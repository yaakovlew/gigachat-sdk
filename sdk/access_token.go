package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/yaakovlew/gigachat-sdk/certificates"

	log "github.com/sirupsen/logrus"
)

const (
	authUrl = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
)

type gigaChatAccessToken struct {
	mu sync.RWMutex

	modelVersion string
	baseToken    string
	clientToken  string
	scopeValue   string

	gigaToken jwtToken

	cert certificates.Certificate
}

type jwtToken struct {
	accessToken string
	expiresAt   int64
}

func newGigaChatToken(cfg GigaChatConfig, cert certificates.Certificate) *gigaChatAccessToken {
	token := &gigaChatAccessToken{
		modelVersion: cfg.Model,
		baseToken:    cfg.BaseToken,
		clientToken:  cfg.ClientToken,
		scopeValue:   cfg.Scope,

		cert: cert,
	}

	if err := token.updateJWT(); err != nil {
		log.Errorf("failed upadte jwt-token err:%v", err)
	}

	go token.refresh()

	return token
}

func (token *gigaChatAccessToken) updateJWT() error {
	// For phys face
	data := url.Values{}
	data.Set("scope", token.scopeValue)

	req, err := http.NewRequest("POST", authUrl, bytes.NewBufferString(data.Encode()))
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

func (token *gigaChatAccessToken) jwtToken() string {
	token.mu.RLock()
	defer token.mu.RUnlock()

	return token.gigaToken.accessToken
}

func (token *gigaChatAccessToken) expiresJWTTime() int64 {
	token.mu.RLock()
	defer token.mu.RUnlock()

	return token.gigaToken.expiresAt
}

func (token *gigaChatAccessToken) refresh() {
	ticker := time.NewTicker(time.Second * time.Duration(time.Unix((token.gigaToken.expiresAt/1000-30), 0).Sub(time.Now()).Seconds()))

	for range ticker.C {
		if err := token.updateJWT(); err != nil {
			log.Errorf("failed refresh jwt-token err:%v", err)
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

func (token *gigaChatAccessToken) model() string {
	token.mu.RLock()
	defer token.mu.RUnlock()

	return token.modelVersion
}
