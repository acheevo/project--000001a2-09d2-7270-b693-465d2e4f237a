// Package managedai calls the Nimbus project's AIaaS connection from the server.
package managedai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const runTimeout = 6*time.Minute + 10*time.Second

type Prompt struct {
	Provider string `json:"provider"`
	Prompt   string `json:"prompt"`
	Model    string `json:"model,omitempty"`
}

type Result struct {
	Provider string `json:"provider"`
	Result   string `json:"result"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// SecureAppSigningKey checks that a starter's public AI route uses a configured key.
func SecureAppSigningKey(value string) bool {
	key := strings.TrimSpace(value)
	return len(key) >= 32 && !strings.HasPrefix(key, "dev-") && !strings.HasPrefix(key, "your-super-secret")
}

func FromEnvironment() *Client {
	url, token := os.Getenv("AIAAS_URL"), os.Getenv("AIAAS_API_TOKEN")
	if url == "" || token == "" {
		return nil
	}
	return &Client{baseURL: strings.TrimRight(url, "/"), token: token, http: &http.Client{
		Timeout:       runTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (c *Client) Run(ctx context.Context, prompt Prompt) (Result, error) {
	var result Result
	body, err := json.Marshal(prompt)
	if err != nil {
		return result, err
	}
	response, err := c.call(ctx, bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("AIaaS returned HTTP %d", response.StatusCode)
	}
	err = json.NewDecoder(response.Body).Decode(&result)
	return result, err
}

func (c *Client) call(ctx context.Context, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/run", body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	return c.http.Do(request)
}

// ServeHTTP belongs behind the application's authorization middleware.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(runTimeout + 15*time.Second))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"could not read prompt"}`)
		return
	}
	response, err := c.call(r.Context(), bytes.NewReader(body))
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `{"error":"could not reach managed AI"}`)
		return
	}
	defer response.Body.Close()
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}
