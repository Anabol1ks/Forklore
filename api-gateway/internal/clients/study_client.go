package clients

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type StudyClient struct {
	baseURL string
	client  *http.Client
}

func NewStudyClient(baseURL string) *StudyClient {
	return &StudyClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *StudyClient) GenerateText(ctx context.Context, mode string, count int, noteText []byte) (int, []byte, error) {
	endpoint, err := url.Parse(c.baseURL + "/generate-text")
	if err != nil {
		return 0, nil, err
	}

	query := endpoint.Query()
	query.Set("mode", mode)
	query.Set("count", strconv.Itoa(count))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(noteText))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, payload, nil
}
