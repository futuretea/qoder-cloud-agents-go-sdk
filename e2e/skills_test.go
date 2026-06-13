//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/skills"
)

func TestE2ESkills(t *testing.T) {
	requireAck(t)
	requireProdOk(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := newClient(t)
	filename := newE2EResourceName(t, "skill") + ".zip"
	data := newMinimalSkillZip(t)

	// Fallback to fixture if dynamic generation somehow produced no data.
	if len(data) == 0 {
		fixture, err := os.ReadFile("e2e/testdata/minimal-skill.zip")
		if err != nil {
			t.Skipf("failed to generate skill zip and read fixture: %v", err)
		}
		data = fixture
	}

	skill, err := createWithRetryValue(ctx, func() (*skills.Skill, error) {
		return c.Skills().Create(ctx, &skills.CreateSkillRequest{
			Filename: filename,
			Data:     data,
			Type:     "custom",
		})
	})
	if err != nil {
		t.Fatalf("failed to create skill: %v", redact(err.Error()))
	}
	recordResource(t, "skill", skill.ID, filename)
	t.Cleanup(func() { safeDelete(t, "skill", skill.ID, filename) })

	got, err := c.Skills().Get(ctx, skill.ID, false)
	if err != nil {
		t.Fatalf("failed to get skill: %v", redact(err.Error()))
	}
	if got.ID != skill.ID {
		t.Fatalf("skill id mismatch: got %q, want %q", got.ID, skill.ID)
	}

	list, err := c.Skills().List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list skills: %v", redact(err.Error()))
	}
	if len(list.Data) == 0 {
		t.Fatal("skills list is empty")
	}

	updated, err := c.Skills().Update(ctx, skill.ID, skills.NewUpdateRequest().WithName("updated-minimal").WithDescription("updated by e2e"))
	if err != nil {
		t.Fatalf("failed to update skill: %v", redact(err.Error()))
	}
	if updated.Name != "updated-minimal" {
		t.Fatalf("skill name after update mismatch: got %q, want %q", updated.Name, "updated-minimal")
	}

	got2, err := c.Skills().Get(ctx, skill.ID, false)
	if err != nil {
		t.Fatalf("failed to get skill after update: %v", redact(err.Error()))
	}
	if got2.Name != "updated-minimal" {
		t.Fatalf("skill name after get mismatch: got %q, want %q", got2.Name, "updated-minimal")
	}

	t.Logf("skill lifecycle passed: %s", skill.ID)
}
