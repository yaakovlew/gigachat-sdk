package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/yaakovlew/gigachat-sdk/certificates"
)

const (
	apiUrl = "https://gigachat.devices.sberbank.ru/api/v1/chat/completions"
)

type GigaChatApi struct {
	token *gigaChatAccessToken
	cert  certificates.Certificate

	statusCode atomic.Int32
}

func NewGigaChatApi(cfg GigaChatConfig, cert certificates.Certificate) *GigaChatApi {
	chat := &GigaChatApi{
		cert:  cert,
		token: newGigaChatToken(cfg, cert),
	}

	return chat
}

func (api *GigaChatApi) Send(messages []Message) (Response, int, error) {
	payload := RequestPayload{
		Model:             api.token.model(),
		Messages:          messages,
		Stream:            false,
		RepetitionPenalty: 1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Response{}, 0, err
	}

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return Response{}, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.token.jwtToken()))

	resp, err := api.cert.HttpClient().Do(req)
	if err != nil {
		return Response{}, 0, err
	}
	defer resp.Body.Close()

	// set last response some error
	api.statusCode.Store(int32(resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		return Response{}, resp.StatusCode, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, resp.StatusCode, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return Response{}, resp.StatusCode, err
	}

	if len(response.Choices) == 0 {
		return Response{}, resp.StatusCode, fmt.Errorf("not valid response")
	}

	return response, resp.StatusCode, nil
}

func (api *GigaChatApi) LastStatusCode() int {
	return int(api.statusCode.Load())
}
