package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type BackendClient struct {
	URL    string
	Token  string
	Client *http.Client
}

func (c BackendClient) Upload(ctx context.Context, batch Batch) error {
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(batch.Payload))
	if err != nil {
		return err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("backend upload failed: %s: %s", resp.Status, string(body))
	}
	return nil
}
