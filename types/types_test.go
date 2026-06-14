package types

import (
	"testing"
)

func TestListParamsToQuery(t *testing.T) {
	tests := []struct {
		name   string
		params ListParams
		want   map[string]string
	}{
		{
			name:   "zero values produce empty query",
			params: ListParams{},
			want:   map[string]string{},
		},
		{
			name:   "limit only",
			params: ListParams{Limit: 10},
			want:   map[string]string{"limit": "10"},
		},
		{
			name:   "after_id only",
			params: ListParams{AfterID: "agent_001"},
			want:   map[string]string{"after_id": "agent_001"},
		},
		{
			name:   "before_id only",
			params: ListParams{BeforeID: "agent_050"},
			want:   map[string]string{"before_id": "agent_050"},
		},
		{
			name:   "all fields set",
			params: ListParams{Limit: 50, AfterID: "cursor_abc", BeforeID: "cursor_xyz"},
			want:   map[string]string{"limit": "50", "after_id": "cursor_abc", "before_id": "cursor_xyz"},
		},
		{
			name:   "limit zero is omitted",
			params: ListParams{Limit: 0, AfterID: "id_123"},
			want:   map[string]string{"after_id": "id_123"},
		},
		{
			name:   "empty string after_id is omitted",
			params: ListParams{Limit: 20, AfterID: "", BeforeID: ""},
			want:   map[string]string{"limit": "20"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.ToQuery()
			if len(got) != len(tt.want) {
				t.Errorf("ToQuery() has %d keys, want %d: got=%v want=%v", len(got), len(tt.want), got, tt.want)
				return
			}
			for k, wantV := range tt.want {
				if got.Get(k) != wantV {
					t.Errorf("ToQuery()[%q] = %q, want %q", k, got.Get(k), wantV)
				}
			}
		})
	}
}

func TestListParamsValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  ListParams
		wantErr bool
	}{
		{name: "zero limit is valid", params: ListParams{Limit: 0}, wantErr: false},
		{name: "limit 1 is valid", params: ListParams{Limit: 1}, wantErr: false},
		{name: "limit 100 is valid", params: ListParams{Limit: 100}, wantErr: false},
		{name: "negative limit", params: ListParams{Limit: -1}, wantErr: true},
		{name: "limit over 100", params: ListParams{Limit: 101}, wantErr: true},
		{name: "limit with after_id", params: ListParams{Limit: 50, AfterID: "id_001"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata Metadata
		wantErr  bool
	}{
		{name: "empty metadata is valid", metadata: Metadata{}, wantErr: false},
		{name: "nil metadata is valid", metadata: nil, wantErr: false},
		{name: "single key-value", metadata: Metadata{"key": "value"}, wantErr: false},
		{name: "16 keys is valid", metadata: makeMetadata(16), wantErr: false},
		{name: "17 keys returns error", metadata: makeMetadata(17), wantErr: true},
		{name: "64-char key is valid", metadata: Metadata{string(make([]byte, 64)): "v"}, wantErr: false},
		{name: "65-char key returns error", metadata: Metadata{string(make([]byte, 65)): "v"}, wantErr: true},
		{name: "512-char value is valid", metadata: Metadata{"k": string(make([]byte, 512))}, wantErr: false},
		{name: "513-char value returns error", metadata: Metadata{"k": string(make([]byte, 513))}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// makeMetadata creates a Metadata map with n entries.
func makeMetadata(n int) Metadata {
	m := make(Metadata, n)
	for i := range n {
		m[string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	return m
}
