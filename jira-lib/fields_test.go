package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFieldValue(t *testing.T) {
	for _, tc := range []struct {
		schema FieldSchema
		raw    string
		want   any
	}{
		{FieldSchema{Type: "string"}, `"text"`, "text"},
		{FieldSchema{Type: "string"}, `null`, nil},
		{FieldSchema{Type: "number"}, `3.0`, 3},
		{FieldSchema{Type: "number"}, `3.5`, 3.5},
		{FieldSchema{Type: "user"}, `{"name":"jo","emailAddress":"jo@x.com","displayName":"Jo"}`, "jo@x.com"},
		{FieldSchema{Type: "user"}, `{"accountId":"abc","displayName":"Jo"}`, "abc"},
		{FieldSchema{Type: "priority"}, `{"id":"3","name":"Medium"}`, "Medium"},
		{FieldSchema{Type: "option"}, `{"id":"1","value":"COM"}`, "COM"},
		{FieldSchema{Type: "issuelink"}, `{"key":"X-1","fields":{"summary":"Parent"}}`, "X-1"},
		{FieldSchema{Type: "any", Custom: "com.atlassian.teams:rm-teams-custom-field-team"}, `{"id":3037,"name":"Team A"}`, "Team A"},
		{FieldSchema{Type: "array", Items: "string"}, `["a","b"]`, []any{"a", "b"}},
		{FieldSchema{Type: "array", Items: "string"}, `null`, []any{}},
		{FieldSchema{Type: "array", Items: "component"}, `[{"name":"c1"},{"name":"c2"}]`, []any{"c1", "c2"}},
	} {
		f := FieldMeta{Name: "F", Schema: tc.schema}
		if got := f.Value(json.RawMessage(tc.raw)); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Value(%v, %s) = %#v, want %#v", tc.schema, tc.raw, got, tc.want)
		}
	}
}

func TestFieldPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/2/user/search":
			if r.URL.Query().Get("username") == "jo@x.com" {
				_, _ = w.Write([]byte(`[{"name":"jo","emailAddress":"jo@x.com"},{"name":"joe","emailAddress":"joe@x.com"}]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case "/rest/teams/1.0/teams/find":
			_, _ = w.Write([]byte(`{"teams":[{"id":1,"title":"Team A-B"},{"id":2,"title":"Team A"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c, err := NewClient(InstanceConfig{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	priority := FieldMeta{Name: "Priority", Schema: FieldSchema{Type: "priority"},
		AllowedValues: []AllowedValue{{ID: "2", Name: "High"}, {ID: "3", Name: "Medium"}}}
	for _, tc := range []struct {
		field FieldMeta
		in    any
		want  any
	}{
		{FieldMeta{Schema: FieldSchema{Type: "string"}}, "  text ", "text"},
		{FieldMeta{Schema: FieldSchema{Type: "string"}}, "", nil},
		{FieldMeta{Schema: FieldSchema{Type: "number"}}, 3, 3},
		{FieldMeta{Schema: FieldSchema{Type: "number"}}, "2.5", 2.5},
		{FieldMeta{Schema: FieldSchema{Type: "date"}}, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), "2026-11-01"},
		{FieldMeta{Schema: FieldSchema{Type: "user"}}, "jo@x.com", map[string]string{"name": "jo"}},
		{FieldMeta{Schema: FieldSchema{Type: "user"}}, nil, nil},
		{priority, "medium", map[string]string{"id": "3"}},
		{FieldMeta{Schema: FieldSchema{Type: "option"}}, "COM", map[string]string{"value": "COM"}},
		{FieldMeta{Schema: FieldSchema{Type: "issuelink"}}, "x-1", map[string]string{"key": "X-1"}},
		{FieldMeta{Schema: FieldSchema{Type: "any", Custom: "com.pyxis.greenhopper.jira:gh-epic-link"}}, "x-2", "X-2"},
		{FieldMeta{Schema: FieldSchema{Type: "any", Custom: "com.atlassian.teams:rm-teams-custom-field-team"}}, "team a", "2"},
		{FieldMeta{Schema: FieldSchema{Type: "array", Items: "string"}}, []any{"a", "", "b"}, []any{"a", "b"}},
		{FieldMeta{Schema: FieldSchema{Type: "array", Items: "string"}}, "solo", []any{"solo"}},
		{FieldMeta{Schema: FieldSchema{Type: "array", Items: "string"}}, nil, []any{}},
	} {
		got, err := c.FieldPayload(context.Background(), tc.field, tc.in)
		if err != nil {
			t.Errorf("FieldPayload(%v, %#v): %v", tc.field.Schema, tc.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("FieldPayload(%v, %#v) = %#v, want %#v", tc.field.Schema, tc.in, got, tc.want)
		}
	}

	for _, tc := range []struct {
		field FieldMeta
		in    any
		err   string
	}{
		{priority, "Bogus", `"Bogus" is not one of: High, Medium`},
		{FieldMeta{Schema: FieldSchema{Type: "number"}}, "lots", "not a number"},
		{FieldMeta{Schema: FieldSchema{Type: "user"}}, "nobody", "no user found"},
		{FieldMeta{Schema: FieldSchema{Type: "string"}}, []any{"a"}, "not a list"},
		{FieldMeta{Schema: FieldSchema{Type: "array", Items: "string", Custom: "com.pyxis.greenhopper.jira:gh-sprint"}}, "1", "isn't supported"},
		{FieldMeta{Schema: FieldSchema{Type: "any", Custom: "com.atlassian.teams:rm-teams-custom-field-team"}}, "Team", "similar teams: Team A-B, Team A"},
	} {
		_, err := c.FieldPayload(context.Background(), tc.field, tc.in)
		if err == nil || !strings.Contains(err.Error(), tc.err) {
			t.Errorf("FieldPayload(%v, %#v) error = %v, want %q", tc.field.Schema, tc.in, err, tc.err)
		}
	}
}
