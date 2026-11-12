package client

import (
	"context"
	"net/url"
	"strconv"
)

// Repo represents a Kosli repo as returned by the API. Kosli records a repo
// when an attestation or trail references it, so repos cannot be created or
// deleted through the API.
type Repo struct {
	// ID is the repo's internal Kosli ID. The tags endpoint identifies repos
	// by this ID because names are not unique across VCS providers.
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Provider    string            `json:"provider"`
	VCSInstance string            `json:"vcs_instance"`
	Tags        map[string]string `json:"tags"`
}

// ReposPage is the paginated envelope returned by the list endpoint.
type ReposPage struct {
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	TotalPages int    `json:"total_pages"`
	TotalCount int    `json:"total_count"`
	Repos      []Repo `json:"repos"`
}

// ListReposOptions are the optional filters for ListRepos. Empty values are
// omitted from the request. Name and Search are mutually exclusive.
type ListReposOptions struct {
	Name        string   // exact repo name
	Search      string   // case-insensitive substring of the repo name
	Provider    string   // VCS provider, e.g. github or gitlab
	VCSInstance string   // VCS host, e.g. gitlab.company.com
	Namespace   string   // '/'-separated namespace prefix, e.g. acme/platform
	Tags        []string // tag key or key:value; keys are ANDed, values of one key ORed
}

// reposPageSize is the API's maximum per_page value.
const reposPageSize = 50

// ListRepos retrieves every repo matching opts, requesting each page of
// GET /api/v2/repos/{org} in turn.
func (c *Client) ListRepos(ctx context.Context, opts *ListReposOptions) ([]Repo, error) {
	all := []Repo{}
	for page := 1; ; page++ {
		query := listReposQuery(opts)
		query.Set("page", strconv.Itoa(page))
		query.Set("per_page", strconv.Itoa(reposPageSize))
		path := apiPath("repos", c.Organization()) + "?" + query.Encode()

		resp, err := c.Get(ctx, path)
		if err != nil {
			return nil, err
		}

		var result ReposPage
		if err := ParseResponse(resp, &result); err != nil {
			return nil, err
		}

		all = append(all, result.Repos...)
		if page >= result.TotalPages || len(result.Repos) == 0 {
			return all, nil
		}
	}
}

func listReposQuery(opts *ListReposOptions) url.Values {
	query := url.Values{}
	if opts == nil {
		return query
	}
	for key, value := range map[string]string{
		"name":         opts.Name,
		"search":       opts.Search,
		"provider":     opts.Provider,
		"vcs_instance": opts.VCSInstance,
		"namespace":    opts.Namespace,
	} {
		if value != "" {
			query.Set(key, value)
		}
	}
	for _, tag := range opts.Tags {
		query.Add("tag", tag)
	}
	return query
}

// GetRepo retrieves a repo by its internal ID via
// GET /api/v2/repos/{org}/{repo_name}?id={id}. The endpoint ignores the name
// segment when id is given, so the ID fills that segment too.
func (c *Client) GetRepo(ctx context.Context, id string) (*Repo, error) {
	path := apiPath("repos", c.Organization(), id) + "?" + url.Values{"id": {id}}.Encode()

	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result Repo
	if err := ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
