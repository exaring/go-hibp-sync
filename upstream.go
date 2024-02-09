package hibpsync

import (
	"fmt"
	"io"
	"net/http"
)

type hibpClient struct {
	endpoint   string
	httpClient *http.Client
}

type hibpResponse struct {
	NotModified bool
	ETag        string
	Data        []byte
}

func (h *hibpClient) RequestRange(rangePrefix, etag string) (*hibpResponse, error) {
	req, err := http.NewRequest("GET", h.endpoint+rangePrefix, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for range %q: %w", rangePrefix, err)
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request for range %q: %w", rangePrefix, err)
	}

	if resp.StatusCode == http.StatusNotModified {
		return &hibpResponse{NotModified: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code requesting range %q: %d", rangePrefix, resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body for range %q: %w", rangePrefix, err)
	}

	return &hibpResponse{
		ETag: resp.Header.Get("ETag"),
		Data: body,
	}, nil
}
