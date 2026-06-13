// Package qoderhttp provides Qoder-specific HTTP helpers built on go-http-client.
// It handles authentication, Qoder error envelope parsing, multipart uploads, and SSE streaming.
package qoderhttp

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"time"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// Config holds the configuration for creating a Qoder HTTP client.
type Config struct {
	BaseURL    string
	Token      string
	Timeout    time.Duration
	HTTPClient httpclient.Doer
}

// NewClient creates an httpclient.Client pre-configured for the Qoder API.
// It automatically injects Bearer token authentication and parses
// Qoder-specific error envelopes.
func NewClient(cfg *Config) httpclient.Client {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	opts := []httpclient.Option{
		httpclient.WithMiddleware(httpclient.AuthMiddleware("Bearer", cfg.Token)),
		httpclient.WithResponseMiddleware(QoderErrorMiddleware),
	}
	if cfg.HTTPClient != nil {
		opts = append(opts, httpclient.WithHTTPClient(cfg.HTTPClient))
	}

	return httpclient.NewClient(&httpclient.Config{
		BaseURL: cfg.BaseURL,
		Timeout: cfg.Timeout,
	}, opts...)
}

// multipartBody builds a multipart/form-data request body with a file field and extra form fields.
// Returns the body bytes and the Content-Type header value.
func multipartBody(fieldName, filename string, data []byte, extraFields map[string]string) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for k, v := range extraFields {
		if err := w.WriteField(k, v); err != nil {
			return nil, "", fmt.Errorf("qoderhttp: write field %s: %w", k, err)
		}
	}

	part, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		return nil, "", fmt.Errorf("qoderhttp: create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, "", fmt.Errorf("qoderhttp: write file data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("qoderhttp: close multipart writer: %w", err)
	}

	return buf.Bytes(), w.FormDataContentType(), nil
}

// ApplyListParams applies cursor-based pagination query parameters to a request.
// It is a convenience helper to reduce duplication across resource packages.
func ApplyListParams(req *httpclient.RequestBuilder, params *types.ListParams) *httpclient.RequestBuilder {
	if params == nil {
		return req
	}
	for k, vs := range params.ToQuery() {
		for _, v := range vs {
			req = req.WithQuery(k, v)
		}
	}
	return req
}

// ApplyIdempotencyKey sets the Idempotency-Key header on the request if a non-empty key is provided.
// It is a convenience helper for Create methods across resource packages.
func ApplyIdempotencyKey(req *httpclient.RequestBuilder, idempotencyKey ...string) *httpclient.RequestBuilder {
	if len(idempotencyKey) > 0 && idempotencyKey[0] != "" {
		req = req.WithHeader("Idempotency-Key", idempotencyKey[0])
	}
	return req
}

// PostMultipart sends a multipart/form-data POST request.
// It builds the multipart body and sends it using the provided client.
// extraHeaders are additional HTTP headers (e.g., Idempotency-Key).
func PostMultipart(ctx context.Context, client httpclient.Client, path string, fieldName, filename string, data []byte, extraFields map[string]string, extraHeaders map[string]string, result interface{}) error {
	body, contentType, err := multipartBody(fieldName, filename, data, extraFields)
	if err != nil {
		return err
	}

	req := client.POST(path).
		WithHeader("Content-Type", contentType).
		WithBody(body).
		WithContext(ctx)
	for k, v := range extraHeaders {
		req = req.WithHeader(k, v)
	}
	return req.Do(result)
}
