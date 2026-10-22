package jira

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAndLookup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jira.yaml")
	data := `projects:
  - project: AAA
    instance: ONE
    teams: [t1, t2]
  - project: BBB
    instance: TWO
instances:
  - instance: ONE
    url: https://example.com/
    username: a@example.com
    token: tok
  - instance: TWO
    url: https://x.atlassian.net/
defaults:
  project: AAA
  team: t1
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	p, err := cfg.Project("")
	if err != nil || p.Project != "AAA" || p.Instance != "ONE" || len(p.Teams) != 2 {
		t.Fatalf("default project: got %+v, %v", p, err)
	}
	if _, err := cfg.Project("ZZZ"); err == nil {
		t.Fatal("expected error for unknown project")
	}

	inst, err := cfg.Instance("TWO")
	if err != nil || inst.URL != "https://x.atlassian.net/" {
		t.Fatalf("instance: got %+v, %v", inst, err)
	}
	if _, err := cfg.Instance("ZZZ"); err == nil {
		t.Fatal("expected error for unknown instance")
	}
}
