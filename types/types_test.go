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
