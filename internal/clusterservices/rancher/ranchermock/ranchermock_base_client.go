package ranchermock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type MockReaderCloser struct {
	io.Reader
}

func (c MockReaderCloser) Close() error {
	return nil
}

type MockBaseClient struct {
	resources map[string]any
}

func (c *MockBaseClient) RoundTrip(r *http.Request) (*http.Response, error) {
	resource, hasResource := c.resources[r.URL.Path]
	if !hasResource {
		return &http.Response{StatusCode: http.StatusNotFound}, nil
	}

	b, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}

	body := MockReaderCloser{
		bytes.NewBuffer(b),
	}

	response := http.Response{
		StatusCode: http.StatusOK,
		Body:       body,
	}

	return &response, nil
}
