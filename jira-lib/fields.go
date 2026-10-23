package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// fieldKind is how a field's values are converted to and from text.
type fieldKind int

const (
	kindUnsupported fieldKind = iota
	kindText
	kindNumber
	kindDate
	kindUser
	kindNamed  // e.g. priority, version, component: {"name": ...}
	kindOption // select list: {"value": ...}
	kindIssue  // e.g. parent: {"key": ...}
	kindEpic   // Epic Link: the epic's key
	kindTeam
)

func (f FieldMeta) kind() (kind fieldKind, array bool) {
	switch f.Schema.Custom {
	case "com.pyxis.greenhopper.jira:gh-epic-link":
		return kindEpic, false
	case "com.pyxis.greenhopper.jira:gh-sprint":
		// Sprints are arrays of strings on Data Center, but can't be set
		// by name.
		return kindUnsupported, true
	case "com.atlassian.teams:rm-teams-custom-field-team",
		"com.atlassian.jira.plugin.system.customfieldtypes:atlassian-team":
		return kindTeam, false
	}
	t := f.Schema.Type
	if t == "array" {
		t, array = f.Schema.Items, true
	}
	switch t {
	case "string":
		return kindText, array
	case "number":
		return kindNumber, array
	case "date", "datetime":
		return kindDate, array
	case "user":
		return kindUser, array
	case "priority", "issuetype", "version", "component", "resolution":
		return kindNamed, array
	case "option":
		return kindOption, array
	case "issuelink":
		return kindIssue, array
	}
	return kindUnsupported, array
}

// Supported reports whether the field's values can be converted by Value
// and FieldPayload.
func (f FieldMeta) Supported() bool {
	k, _ := f.kind()
	return k != kindUnsupported
}

// IsArray reports whether the field holds a list of values.
func (f FieldMeta) IsArray() bool {
	_, array := f.kind()
	return array
}

// Value converts a field's raw JSON value into a plain value for editing: a
// string, an int or float64, a []any of those, or nil if the field is empty.
// Users are shown by email address (or username, or account ID), objects by
// name or value, linked issues by key and teams by name.
func (f FieldMeta) Value(raw json.RawMessage) any {
	kind, array := f.kind()
	var v any
	if json.Unmarshal(raw, &v) != nil || v == nil {
		if array {
			return []any{}
		}
		return nil
	}
	if !array {
		return plainValue(kind, v)
	}
	list := []any{}
	if items, ok := v.([]any); ok {
		for _, item := range items {
			if p := plainValue(kind, item); p != nil {
				list = append(list, p)
			}
		}
	}
	return list
}

func plainValue(kind fieldKind, v any) any {
	switch v := v.(type) {
	case string:
		return v
	case float64:
		if v == math.Trunc(v) && math.Abs(v) < 1<<53 {
			return int(v)
		}
		return v
	case map[string]any:
		keys := []string{"name", "value"}
		switch kind {
		case kindUser:
			keys = []string{"emailAddress", "name", "accountId"}
		case kindIssue:
			keys = []string{"key"}
		}
		for _, k := range keys {
			if s, ok := v[k].(string); ok && s != "" {
				return s
			}
		}
	}
	return nil
}

// IsEmpty reports whether a plain value (as returned by Value or decoded
// from YAML) is empty: nil, a blank string or an empty list.
func IsEmpty(v any) bool {
	switch v := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	}
	return false
}

// FieldPayload converts a plain value (as returned by Value, or decoded from
// YAML) into the value the REST API expects for the field. An empty value
// clears the field. Users and teams are looked up by name, and named values
// are checked against the field's allowed values.
func (c *Client) FieldPayload(ctx context.Context, f FieldMeta, v any) (any, error) {
	kind, array := f.kind()
	if kind == kindUnsupported {
		return nil, fmt.Errorf("%s: setting this field isn't supported", f.Name)
	}
	if !array {
		if IsEmpty(v) {
			return nil, nil
		}
		return c.itemPayload(ctx, f, kind, v)
	}
	items, ok := v.([]any)
	if !ok && !IsEmpty(v) {
		items = []any{v}
	}
	list := []any{}
	for _, item := range items {
		if IsEmpty(item) {
			continue
		}
		p, err := c.itemPayload(ctx, f, kind, item)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

func (c *Client) itemPayload(ctx context.Context, f FieldMeta, kind fieldKind, v any) (any, error) {
	if _, ok := v.([]any); ok {
		return nil, fmt.Errorf("%s: expected a single value, not a list", f.Name)
	}
	if _, ok := v.(map[string]any); ok {
		return nil, fmt.Errorf("%s: expected a single value, not a mapping", f.Name)
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	switch kind {
	case kindText:
		return s, nil
	case kindNumber:
		switch v := v.(type) {
		case int, float64:
			return v, nil
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: %q is not a number", f.Name, s)
		}
		return n, nil
	case kindDate:
		// YAML decodes unquoted dates as timestamps.
		if t, ok := v.(time.Time); ok {
			if f.Schema.Type == "datetime" {
				return t.Format(TimeFormat), nil
			}
			return t.Format(time.DateOnly), nil
		}
		return s, nil
	case kindUser:
		u, err := c.FindUser(ctx, s)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		return c.UserRef(u), nil
	case kindNamed, kindOption:
		if len(f.AllowedValues) == 0 {
			if kind == kindOption {
				return map[string]string{"value": s}, nil
			}
			return map[string]string{"name": s}, nil
		}
		var choices []string
		for _, a := range f.AllowedValues {
			if strings.EqualFold(a.Text(), s) {
				return map[string]string{"id": a.ID}, nil
			}
			choices = append(choices, a.Text())
		}
		return nil, fmt.Errorf("%s: %q is not one of: %s", f.Name, s, strings.Join(choices, ", "))
	case kindIssue:
		return map[string]string{"key": strings.ToUpper(s)}, nil
	case kindEpic:
		return strings.ToUpper(s), nil
	case kindTeam:
		id, err := c.FindTeam(ctx, f, s)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		return id, nil
	}
	return nil, fmt.Errorf("%s: unsupported field", f.Name)
}
