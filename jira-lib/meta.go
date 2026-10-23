package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// FieldSchema describes the type of a field's value.
type FieldSchema struct {
	// Type is e.g. "string", "number", "user", "option" or "array".
	Type string `json:"type"`
	// Items is the type of each element when Type is "array".
	Items string `json:"items"`
	// Custom identifies the custom field type, for custom fields.
	Custom string `json:"custom"`
}

// AllowedValue is one of the values a field may be set to. Most kinds of
// value have a Name; select list options have a Value instead.
type AllowedValue struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Text returns the value's display text.
func (v AllowedValue) Text() string {
	if v.Name != "" {
		return v.Name
	}
	return v.Value
}

// FieldMeta describes a field that can be set when creating or editing an
// issue.
type FieldMeta struct {
	ID              string         `json:"fieldId"`
	Name            string         `json:"name"`
	Required        bool           `json:"required"`
	HasDefaultValue bool           `json:"hasDefaultValue"`
	Schema          FieldSchema    `json:"schema"`
	AllowedValues   []AllowedValue `json:"allowedValues"`
}

// EditMeta returns the fields that can be edited on an issue, keyed by field
// ID.
func (c *Client) EditMeta(ctx context.Context, key string) (map[string]FieldMeta, error) {
	var meta struct {
		Fields map[string]FieldMeta `json:"fields"`
	}
	if err := c.Do(ctx, http.MethodGet, "rest/api/2/issue/"+url.PathEscape(key)+"/editmeta", nil, nil, &meta); err != nil {
		return nil, err
	}
	// Cloud doesn't include fieldId in edit metadata.
	for id, f := range meta.Fields {
		f.ID = id
		meta.Fields[id] = f
	}
	return meta.Fields, nil
}

// IssueTypeMeta is an issue type that can be created in a project.
type IssueTypeMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IssueTypes returns the issue types that can be created in a project.
func (c *Client) IssueTypes(ctx context.Context, project string) ([]IssueTypeMeta, error) {
	var types []IssueTypeMeta
	err := c.pagedMeta(ctx, "rest/api/2/issue/createmeta/"+url.PathEscape(project)+"/issuetypes", "issueTypes", func(raw json.RawMessage) error {
		var page []IssueTypeMeta
		if err := json.Unmarshal(raw, &page); err != nil {
			return err
		}
		types = append(types, page...)
		return nil
	})
	return types, err
}

// CreateMeta returns the fields that can be set when creating an issue of
// the given type ID in a project, keyed by field ID.
func (c *Client) CreateMeta(ctx context.Context, project, issueTypeID string) (map[string]FieldMeta, error) {
	fields := map[string]FieldMeta{}
	err := c.pagedMeta(ctx, "rest/api/2/issue/createmeta/"+url.PathEscape(project)+"/issuetypes/"+url.PathEscape(issueTypeID), "fields", func(raw json.RawMessage) error {
		var page []FieldMeta
		if err := json.Unmarshal(raw, &page); err != nil {
			return err
		}
		for _, f := range page {
			fields[f.ID] = f
		}
		return nil
	})
	return fields, err
}

// pagedMeta fetches every page of a createmeta list, passing each page's
// items to add. Data Center returns the items under "values"; Cloud under
// cloudKey.
func (c *Client) pagedMeta(ctx context.Context, path, cloudKey string, add func(json.RawMessage) error) error {
	for startAt := 0; ; {
		q := url.Values{"startAt": {strconv.Itoa(startAt)}, "maxResults": {"200"}}
		var page map[string]json.RawMessage
		if err := c.Do(ctx, http.MethodGet, path, q, nil, &page); err != nil {
			return err
		}
		items := page["values"]
		if items == nil {
			items = page[cloudKey]
		}
		var list []json.RawMessage
		if err := json.Unmarshal(items, &list); err != nil {
			return fmt.Errorf("%s: unexpected response: %w", path, err)
		}
		if err := add(items); err != nil {
			return err
		}
		startAt += len(list)
		var total int
		var isLast bool
		_ = json.Unmarshal(page["total"], &total)
		_ = json.Unmarshal(page["isLast"], &isLast)
		if len(list) == 0 || isLast || startAt >= total {
			return nil
		}
	}
}

// FindField finds a field by ID or display name, case-insensitively. An ID
// match wins; a name shared by several fields is an error.
func FindField(fields map[string]FieldMeta, name string) (FieldMeta, error) {
	var matches []FieldMeta
	for id, f := range fields {
		if strings.EqualFold(id, name) {
			return f, nil
		}
		if strings.EqualFold(f.Name, name) {
			matches = append(matches, f)
		}
	}
	switch len(matches) {
	case 0:
		return FieldMeta{}, fmt.Errorf("unknown field %q", name)
	case 1:
		return matches[0], nil
	}
	var ids []string
	for _, f := range matches {
		ids = append(ids, f.ID)
	}
	slices.Sort(ids)
	return FieldMeta{}, fmt.Errorf("field name %q is ambiguous, use one of the IDs: %s", name, strings.Join(ids, ", "))
}
