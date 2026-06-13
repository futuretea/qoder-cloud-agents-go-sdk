package qoder

import (
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	client := New("test-token")
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.http == nil {
		t.Fatal("expected internal http client to be non-nil")
	}
}

func TestWithHTTPClient(t *testing.T) {
	customHC := &http.Client{}
	client := New("test-token", WithHTTPClient(customHC))
	if client.http == nil {
		t.Fatal("expected internal http client to be non-nil")
	}
	// Verify the custom *http.Client is actually stored and used.
	if client.httpClient != customHC {
		t.Error("expected custom HTTP client to be stored")
	}
}

func TestWithBaseURL(t *testing.T) {
	customURL := "https://custom.example.com/api"
	client := New("test-token", WithBaseURL(customURL))
	if client.http == nil {
		t.Fatal("expected internal http client to be non-nil")
	}
	// Verify the custom base URL is actually stored.
	if client.baseURL != customURL {
		t.Errorf("expected baseURL %q, got %q", customURL, client.baseURL)
	}
}

func TestWithBaseURLPreservesHTTPClient(t *testing.T) {
	customHC := &http.Client{}
	customURL := "https://custom.example.com/api"
	client := New("test-token", WithHTTPClient(customHC), WithBaseURL(customURL))
	if client.httpClient != customHC {
		t.Error("WithBaseURL should preserve the custom HTTP client set by WithHTTPClient")
	}
	if client.baseURL != customURL {
		t.Errorf("expected baseURL %q, got %q", customURL, client.baseURL)
	}
}

func TestWithHTTPClientPreservesToken(t *testing.T) {
	client := New("test-token", WithHTTPClient(&http.Client{}))
	if client.token != "test-token" {
		t.Errorf("expected token to be preserved, got %q", client.token)
	}
}

func TestResourceAccessors(t *testing.T) {
	client := New("test-token")

	t.Run("Agents", func(t *testing.T) {
		a := client.Agents()
		if a == nil {
			t.Error("expected Agents to be non-nil")
		}
		// Verify caching: same instance on repeated calls
		if client.Agents() != a {
			t.Error("expected Agents to be cached")
		}
	})

	t.Run("Environments", func(t *testing.T) {
		e := client.Environments()
		if e == nil {
			t.Error("expected Environments to be non-nil")
		}
	})

	t.Run("Sessions", func(t *testing.T) {
		s := client.Sessions()
		if s == nil {
			t.Error("expected Sessions to be non-nil")
		}
	})

	t.Run("Events", func(t *testing.T) {
		e := client.Events()
		if e == nil {
			t.Error("expected Events to be non-nil")
		}
	})

	t.Run("Files", func(t *testing.T) {
		f := client.Files()
		if f == nil {
			t.Error("expected Files to be non-nil")
		}
	})

	t.Run("Vaults", func(t *testing.T) {
		v := client.Vaults()
		if v == nil {
			t.Error("expected Vaults to be non-nil")
		}
	})

	t.Run("Skills", func(t *testing.T) {
		s := client.Skills()
		if s == nil {
			t.Error("expected Skills to be non-nil")
		}
	})

	t.Run("MemoryStores", func(t *testing.T) {
		m := client.MemoryStores()
		if m == nil {
			t.Error("expected MemoryStores to be non-nil")
		}
	})

	t.Run("Models", func(t *testing.T) {
		m := client.Models()
		if m == nil {
			t.Error("expected Models to be non-nil")
		}
	})
}
