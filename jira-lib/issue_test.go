package jira

import "testing"

func TestResolveIssue(t *testing.T) {
	cfg := &Config{
		Projects: []ProjectConfig{{Project: "AAA", Instance: "ONE"}, {Project: "BBB", Instance: "TWO"}},
		Instances: []InstanceConfig{
			{Instance: "ONE", URL: "https://one.example.com:8443/"},
			{Instance: "TWO", URL: "https://two.example.com/jira/"},
		},
		Defaults: Defaults{Project: "AAA"},
	}
	for _, tc := range []struct {
		ref, instance, key string
	}{
		{"bbb-12", "TWO", "BBB-12"},
		{"ZZZ-1", "ONE", "ZZZ-1"},
		{"https://one.example.com:8443/browse/AAA-5", "ONE", "AAA-5"},
		{"https://TWO.example.com/jira/browse/bbb-7/", "TWO", "BBB-7"},
		{"https://one.example.com:8443/secure/RapidBoard.jspa?rapidView=1&selectedIssue=AAA-9", "ONE", "AAA-9"},
	} {
		inst, key, err := cfg.ResolveIssue(tc.ref)
		if err != nil || inst.Instance != tc.instance || key != tc.key {
			t.Errorf("ResolveIssue(%q) = %v, %q, %v; want %s, %q", tc.ref, inst, key, err, tc.instance, tc.key)
		}
	}
	for _, ref := range []string{"AAA", "AAA-", "https://other.example.com/browse/AAA-1", "https://two.example.com/browse/BBB-1", "https://one.example.com:8443/projects"} {
		if _, _, err := cfg.ResolveIssue(ref); err == nil {
			t.Errorf("ResolveIssue(%q): expected error", ref)
		}
	}
}
