package qoderhttp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// maxSSELineSize is the maximum size of a single SSE line (field + value).
// The default bufio.Scanner limit is 64 KB; we increase it to 1 MB to
// accommodate large data payloads (e.g., base64-encoded artifacts).
const maxSSELineSize = 1 << 20 // 1 MB

// maxSSEEventSize is the maximum total size of a single SSE event (sum of all
// data lines). It prevents unbounded memory growth from a malicious or buggy
// server that emits millions of data lines without an empty-line terminator.
const maxSSEEventSize = 16 << 20 // 16 MB

// SSEEvent represents a parsed Server-Sent Event.
type SSEEvent struct {
	ID    string
	Event string
	Data  []byte
}

// SSEStream parses SSE events from an HTTP response body.
type SSEStream struct {
	scanner   *bufio.Scanner
	body      io.ReadCloser
	closeOnce sync.Once
	closeErr  error

	// cachedEvent and cachedErr hold a parsed result that was completed
	// concurrently with a previous context cancellation, preventing silent
	// event loss when the select in Next() non-deterministically picks
	// ctx.Done() over the parse result channel.
	cachedEvent *SSEEvent
	cachedErr   error
	cachedMu    sync.Mutex
}

// NewSSEStream creates a new SSE stream from an HTTP response.
// The response must have been obtained with Accept: text/event-stream.
func NewSSEStream(resp *http.Response) *SSEStream {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, maxSSELineSize), maxSSELineSize)
	return &SSEStream{
		scanner: scanner,
		body:    resp.Body,
	}
}

// Next reads the next SSE event, blocking until one arrives or the stream ends.
// Returns nil, io.EOF when the stream is complete.
//
// The context can be used to cancel a blocking read. When the context is
// cancelled, Next closes the underlying response body to unblock the scanner
// and returns ctx.Err(). A successfully parsed event that completes
// concurrently with cancellation is not lost — it is cached and returned on
// the next call to Next().
func (s *SSEStream) Next(ctx context.Context) (*SSEEvent, error) {
	// Return a cached event from a previous context-cancelled call.
	s.cachedMu.Lock()
	if s.cachedEvent != nil || s.cachedErr != nil {
		evt, err := s.cachedEvent, s.cachedErr
		s.cachedEvent = nil
		s.cachedErr = nil
		s.cachedMu.Unlock()
		return evt, err
	}
	s.cachedMu.Unlock()

	type parseResult struct {
		evt *SSEEvent
		err error
	}
	ch := make(chan parseResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- parseResult{nil, fmt.Errorf("sse: panic in parseNext: %v", r)}
			}
		}()
		evt, err := s.parseNext()
		ch <- parseResult{evt, err}
	}()

	select {
	case <-ctx.Done():
		// Close the body to unblock the scanner goroutine.
		// Uses Close() (idempotent via sync.Once) so a subsequent
		// defer stream.Close() by the caller is always safe.
		_ = s.Close()
		// Drain the channel to capture any concurrently-completed parse.
		// Close() unblocks the scanner, guaranteeing the goroutine sends.
		r := <-ch
		s.cachedMu.Lock()
		s.cachedEvent = r.evt
		s.cachedErr = r.err
		s.cachedMu.Unlock()
		return nil, ctx.Err()
	case r := <-ch:
		return r.evt, r.err
	}
}

// parseNext performs the blocking SSE parse loop. It is called from Next
// in a separate goroutine so that context cancellation can interrupt it.
func (s *SSEStream) parseNext() (*SSEEvent, error) {
	var evt SSEEvent

	for {
		if !s.scanner.Scan() {
			if err := s.scanner.Err(); err != nil {
				return nil, fmt.Errorf("sse: scan error: %w", err)
			}
			return nil, io.EOF
		}

		line := s.scanner.Text()

		// Empty line signals end of event
		if line == "" {
			if evt.Data != nil || evt.Event != "" {
				return &evt, nil
			}
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field: value per SSE spec.
		// If the line contains a colon, the field is before the colon and the
		// value is after (with one leading space stripped if present).
		// If the line has no colon, the entire line is the field name and the
		// value is the empty string.
		field, value := line, ""
		if colonIdx := strings.Index(line, ":"); colonIdx >= 0 {
			field = line[:colonIdx]
			value = line[colonIdx+1:]
			value = strings.TrimPrefix(value, " ")
		}

		switch field {
		case "id":
			evt.ID = value
		case "event":
			evt.Event = value
		case "data":
			if evt.Data != nil {
				evt.Data = append(evt.Data, '\n')
			}
			evt.Data = append(evt.Data, value...)
			if len(evt.Data) > maxSSEEventSize {
				return nil, fmt.Errorf("sse: event exceeds maximum size of %d bytes", maxSSEEventSize)
			}
		}
	}
}

// Close closes the underlying response body. It is safe to call multiple times.
func (s *SSEStream) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.body.Close()
	})
	return s.closeErr
}
