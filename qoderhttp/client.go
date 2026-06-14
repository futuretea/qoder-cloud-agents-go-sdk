// Package qoderhttp provides Qoder-specific HTTP helpers built on go-http-client.
// It handles authentication, Qoder error envelope parsing, multipart uploads, and SSE streaming.
package qoderhttp

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"sort"
	"strings"
	"time"
	"unicode"

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

	// Write extra fields in deterministic order to make request bodies stable
	// across calls and easier to test or sign.
	keys := make([]string, 0, len(extraFields))
	for k := range extraFields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := w.WriteField(k, extraFields[k]); err != nil {
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

// ValidateID returns an error if the given resource ID is empty or contains
// path traversal sequences. It provides defense-in-depth protection when
// IDs are embedded in URL paths.
//
// Both literal and URL-encoded traversal sequences are rejected
// (e.g., %2F for /, %5C for \, %2E%2E for .., %00 for null byte).
func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("qoderhttp: resource ID must not be empty")
	}
	// Check literal path traversal and URL-breaking characters.
	if strings.ContainsAny(id, "/\\#?") || strings.Contains(id, "..") {
		return fmt.Errorf("qoderhttp: resource ID contains invalid characters: %q", id)
	}
	// Reject whitespace and control characters that are invalid in URLs.
	for _, r := range id {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("qoderhttp: resource ID contains invalid characters: %q", id)
		}
	}
	// Check URL-encoded dangerous sequences (case-insensitive).
	// %2f = /, %5c = \, %2e = ., %00 = null byte.
	// %2e (encoded dot) is rejected unconditionally — there is no legitimate
	// use for an encoded dot in a resource ID, and even a single %2e combined
	// with a literal dot can form .. after URL decoding.
	// %00 (null byte) can cause path truncation in some HTTP frameworks
	// after URL decoding.
	lower := strings.ToLower(id)
	if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") ||
		strings.Contains(lower, "%2e") || strings.Contains(lower, "%00") {
		return fmt.Errorf("qoderhttp: resource ID contains invalid characters: %q", id)
	}
	return nil
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
