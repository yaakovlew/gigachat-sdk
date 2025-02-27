package certificates

import (
	"crypto/tls"
	"net/http"
	"os"
)

type Certificate interface {
	HttpClient() *http.Client
}

type Certs struct {
	path string

	httpClient *http.Client
}

func NewCerts(path string) (*Certs, error) {
	if err := os.Setenv("NODE_EXTRA_CA_CERTS", path); err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Certs{
		path:       path,
		httpClient: client,
	}, nil
}

func (c *Certs) HttpClient() *http.Client {
	return c.httpClient
}
