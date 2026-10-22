package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to the JIRA REST API of a single instance.
type Client struct {
	Instance   InstanceConfig
	HTTPClient *http.Client
	baseURL    *url.URL
}

// NewClient creates a Client for the given instance config.
func NewClient(inst InstanceConfig) (*Client, error) {
	u, err := url.Parse(inst.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing url for instance %s: %w", inst.Instance, err)
	}
	return &Client{
		Instance:   inst,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    u,
	}, nil
}

// NewClientForProject loads ~/.jira.yaml and creates a Client for the
// instance hosting the named project (or the default project if name is
// empty). The resolved project config is returned alongside the client.
func NewClientForProject(name string) (*Client, *ProjectConfig, error) {
	cfg, err := LoadConfig("")
	if err != nil {
		return nil, nil, err
	}
	p, err := cfg.Project(name)
	if err != nil {
		return nil, nil, err
	}
	inst, err := cfg.Instance(p.Instance)
	if err != nil {
		return nil, nil, fmt.Errorf("project %s: %w", p.Project, err)
	}
	c, err := NewClient(*inst)
	if err != nil {
		return nil, nil, err
	}
	return c, p, nil
}

// isCloud reports whether the instance is Atlassian Cloud, which uses basic
// auth with an API token, rather than Data Center, which uses a bearer PAT.
func (c *Client) isCloud() bool {
	return strings.HasSuffix(c.baseURL.Hostname(), ".atlassian.net")
}

// Do sends a request to the REST API path (e.g. "rest/api/2/myself") with
// optional query parameters, and decodes a JSON response into out, if out is
// non-nil.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body io.Reader, out any) error {
	u := c.baseURL.JoinPath(path)
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.isCloud() {
		req.SetBasicAuth(c.Instance.Username, c.Instance.Token)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.Instance.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s %s: %s: %s", method, u, resp.Status, strings.TrimSpace(string(msg)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// User is a JIRA user.
type User struct {
	Name         string `json:"name"`
	AccountID    string `json:"accountId"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
}

// Myself returns the currently authenticated user.
func (c *Client) Myself(ctx context.Context) (*User, error) {
	var u User
	if err := c.Do(ctx, http.MethodGet, "rest/api/2/myself", nil, nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}
