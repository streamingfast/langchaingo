package langsmith

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	apiKey string
	apiURL string
	webURL *string

	hideInputs  bool
	hideOutputs bool

	httpClient *http.Client
}

func NewClient(options ...ClientOption) (*Client, error) {
	c := &Client{
		apiKey:      os.Getenv("LANGCHAIN_API_KEY"),
		apiURL:      envOr("LANGCHAIN_ENDPOINT", "https://api.smith.langchain.com"),
		hideInputs:  envOr("LANGCHAIN_HIDE_INPUTS", "false") == "true",
		hideOutputs: envOr("LANGCHAIN_HIDE_OUTPUTS", "false") == "true",
		webURL:      nil,

		httpClient: &http.Client{
			Timeout: 4 * time.Second,
		},
	}

	for _, option := range options {
		option.apply(c)
	}

	// Sanitization(s)
	c.apiURL = strings.TrimSuffix(c.apiURL, "/")

	// Validation(s)
	if len(c.apiKey) == 0 {
		return nil, ErrMissingAPIKey
	}

	return c, nil
}

func (c *Client) CreateRun(ctx context.Context, run *RunCreate) error {
	// FIXME: Add back getRuntimeEnv logic, for now we assume RunCreate will have populated the fields correctly

	if c.hideInputs {
		run.Inputs = nil
	}

	if c.hideOutputs {
		run.Outputs = nil
	}

	body, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}

	return c.executeHTTPRequest(ctx, "POST", "/runs", nil, bytes.NewBuffer(body))
}

func (c *Client) UpdateRun(ctx context.Context, runID string, run *RunUpdate) error {
	if !isValidUUID(runID) {
		return ErrInvalidUUID
	}

	if c.hideInputs {
		run.Inputs = nil
	}

	if c.hideOutputs {
		run.Outputs = nil
	}

	body, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}

	return c.executeHTTPRequest(ctx, "PATCH", fmt.Sprintf("/runs/%s", runID), nil, bytes.NewBuffer(body))
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func (c *Client) executeHTTPRequest(ctx context.Context, method string, path string, query url.Values, body *bytes.Buffer) error {
	if path[0] != '/' {
		path = "/" + path
	}

	callURL := c.apiURL + path
	if len(query) > 0 {
		callURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, callURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return NewLangSmitAPIErrorFromHTTP(req, resp)
	}

	return nil
}

func (c *Client) getHostURL() string {
	if c.webURL != nil {
		return *c.webURL
	}

	if isLocalhost(c.apiURL) {
		return "http://localhost"
	}

	if strings.Contains(c.apiURL, "/api") {
		return strings.Replace(c.apiURL, "/api", "", -1)
	}

	if strings.Contains(strings.Split(c.apiURL, ".")[0], "dev") {
		return "https://dev.smith.langchain.com"
	}

	return "https://smith.langchain.com"

}
