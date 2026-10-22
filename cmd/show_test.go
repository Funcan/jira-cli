package cmd

import (
	"testing"
	"time"
)

func TestFieldText(t *testing.T) {
	created := time.Date(2026, 8, 1, 16, 29, 3, 0, time.UTC).Local().Format("2006-01-02 15:04")
	for raw, want := range map[string]string{
		`null`:                               "",
		`"plain"`:                            "plain",
		`"2026-08-01T16:29:03.233+0000"`:     created,
		`3.5`:                                "3.5",
		`[]`:                                 "",
		`["a","b"]`:                          "a, b",
		`{"displayName":"Jo","name":"jo@x"}`: "Jo",
		`[{"name":"v1"},{"name":"v2"}]`:      "v1, v2",
		`{"value":"Red","id":"1"}`:           "Red",
		`{"key":"X-1","fields":{"summary":"Parent"}}`: "X-1 Parent",
	} {
		if got := fieldText([]byte(raw)); got != want {
			t.Errorf("fieldText(%s) = %q, want %q", raw, got, want)
		}
	}
}
