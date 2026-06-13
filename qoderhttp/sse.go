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
// and returns ctx.Err(). Note that if a parse completes at approximately the
// same time as context cancellation, the select statement may
// non-deterministically return ctx.Err() even though an event was available.
// Callers should treat context cancellation as a terminal signal and not
// expect further events after cancellation.
func (s *SSEStream) Next(ctx context.Context) (*SSEEvent, error) {
	type parseResult struct {
		evt *SSEEvent
		err error
	}
	ch := make(chan parseResult, 1)

	go func() {
		evt, err := s.parseNext()
		ch <- parseResult{evt, err}
	}()

	select {
	case <-ctx.Done():
		// Close the body to unblock the scanner goroutine.
		// Uses Close() (idempotent via sync.Once) so a subsequent
		// defer stream.Close() by the caller is always safe.
		s.Close()
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
			if strings.HasPrefix(value, " ") {
				value = value[1:]
			}
		}

		switch field {
		case "id":
			evt.ID = value
		case "event":
			evt.Event = value
		case "data":
			if evt.Data != nil {
				evt.Data = append(evt.Data, '\n')
				evt.Data = append(evt.Data, value...)
			} else {
				evt.Data = []byte(value)
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
