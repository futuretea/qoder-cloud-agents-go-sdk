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
		if client.Environments() != e {
			t.Error("expected Environments to be cached")
		}
	})

	t.Run("Sessions", func(t *testing.T) {
		s := client.Sessions()
		if s == nil {
			t.Error("expected Sessions to be non-nil")
		}
		if client.Sessions() != s {
			t.Error("expected Sessions to be cached")
		}
	})

	t.Run("Events", func(t *testing.T) {
		e := client.Events()
		if e == nil {
			t.Error("expected Events to be non-nil")
		}
		if client.Events() != e {
			t.Error("expected Events to be cached")
		}
	})

	t.Run("Files", func(t *testing.T) {
		f := client.Files()
		if f == nil {
			t.Error("expected Files to be non-nil")
		}
		if client.Files() != f {
			t.Error("expected Files to be cached")
		}
	})

	t.Run("Vaults", func(t *testing.T) {
		v := client.Vaults()
		if v == nil {
			t.Error("expected Vaults to be non-nil")
		}
		if client.Vaults() != v {
			t.Error("expected Vaults to be cached")
		}
	})

	t.Run("Skills", func(t *testing.T) {
		s := client.Skills()
		if s == nil {
			t.Error("expected Skills to be non-nil")
		}
		if client.Skills() != s {
			t.Error("expected Skills to be cached")
		}
	})

	t.Run("MemoryStores", func(t *testing.T) {
		m := client.MemoryStores()
		if m == nil {
			t.Error("expected MemoryStores to be non-nil")
		}
		if client.MemoryStores() != m {
			t.Error("expected MemoryStores to be cached")
		}
	})

	t.Run("Models", func(t *testing.T) {
		m := client.Models()
		if m == nil {
			t.Error("expected Models to be non-nil")
		}
		if client.Models() != m {
			t.Error("expected Models to be cached")
		}
	})
}

func TestResourceAccessors_Concurrent(_ *testing.T) {
	client := New("test-token")

	// Access all resource accessors concurrently to verify sync.Once safety.
	// -race will flag any data races.
	done := make(chan struct{})
	accessors := []func(){
		func() { client.Agents() },
		func() { client.Environments() },
		func() { client.Sessions() },
		func() { client.Events() },
		func() { client.Files() },
		func() { client.Vaults() },
		func() { client.Skills() },
		func() { client.MemoryStores() },
		func() { client.Models() },
	}

	for _, fn := range accessors {
		go func(f func()) {
			f()
			done <- struct{}{}
		}(fn)
	}

	for range accessors {
		<-done
	}
}
