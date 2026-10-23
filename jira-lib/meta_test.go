package jira

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateMetaPaging(t *testing.T) {
	// Data Center pages under "values"; Cloud lists fields under "fields".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path + "?" + r.URL.Query().Get("startAt") {
		case "/rest/api/2/issue/createmeta/DC/issuetypes/1?0":
			_, _ = w.Write([]byte(`{"total":2,"values":[{"fieldId":"summary","name":"Summary","required":true}]}`))
		case "/rest/api/2/issue/createmeta/DC/issuetypes/1?1":
			_, _ = w.Write([]byte(`{"total":2,"values":[{"fieldId":"labels","name":"Labels"}]}`))
		case "/rest/api/2/issue/createmeta/CL/issuetypes/1?0":
			_, _ = w.Write([]byte(`{"total":1,"fields":[{"fieldId":"summary","name":"Summary"}]}`))
		default:
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c, err := NewClient(InstanceConfig{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	fields, err := c.CreateMeta(context.Background(), "DC", "1")
	if err != nil || len(fields) != 2 || !fields["summary"].Required || fields["labels"].Name != "Labels" {
		t.Errorf("DC CreateMeta = %+v, %v", fields, err)
	}
	fields, err = c.CreateMeta(context.Background(), "CL", "1")
	if err != nil || len(fields) != 1 || fields["summary"].Name != "Summary" {
		t.Errorf("Cloud CreateMeta = %+v, %v", fields, err)
	}
}

func TestFindField(t *testing.T) {
	fields := map[string]FieldMeta{
		"summary":           {ID: "summary", Name: "Summary"},
		"customfield_1":     {ID: "customfield_1", Name: "Team"},
		"customfield_2":     {ID: "customfield_2", Name: "Team"},
		"customfield_10006": {ID: "customfield_10006", Name: "Story Points"},
	}
	for name, want := range map[string]string{
		"summary": "summary", "SUMMARY": "summary", "story points": "customfield_10006", "customfield_2": "customfield_2",
	} {
		if f, err := FindField(fields, name); err != nil || f.ID != want {
			t.Errorf("FindField(%q) = %q, %v, want %q", name, f.ID, err, want)
		}
	}
	if _, err := FindField(fields, "team"); err == nil || !strings.Contains(err.Error(), "customfield_1, customfield_2") {
		t.Errorf("FindField(team) error = %v, want ambiguity", err)
	}
	if _, err := FindField(fields, "nope"); err == nil {
		t.Error("FindField(nope) succeeded")
	}
}
