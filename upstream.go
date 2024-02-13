package hibpsync

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type hibpClient struct {
	endpoint   string
	httpClient *http.Client
	maxRetries int
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

	var mErr error

	for i := 0; i < 1+h.maxRetries; i++ {
		resp, err := h.request(req)
		if err == nil {
			return resp, nil
		}

		// TODO: Log error with debug level

		mErr = errors.Join(mErr, err)
	}

	return nil, fmt.Errorf("requesting range %q: %w", rangePrefix, mErr)
}

func (h *hibpClient) request(req *http.Request) (*hibpResponse, error) {
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode == http.StatusNotModified {
		return &hibpResponse{NotModified: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &hibpResponse{
		ETag: resp.Header.Get("ETag"),
		Data: body,
	}, nil
}
