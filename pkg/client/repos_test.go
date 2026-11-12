package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func newRepoTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	c, err := NewClient("test-token", "test-org", WithBaseURL(server.URL), WithAPIPath(""))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c
}

func TestListRepos_FollowsPagination(t *testing.T) {
	c := newRepoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repos/test-org" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("per_page"); got != "50" {
			t.Errorf("expected per_page=50, got %q", got)
		}

		page := ReposPage{PerPage: 50, TotalPages: 2, TotalCount: 3}
		switch r.URL.Query().Get("page") {
		case "1":
			page.Page = 1
			page.Repos = []Repo{
				{ID: "id-1", Name: "acme/api", URL: "https://github.com/acme/api", Provider: "github", Tags: map[string]string{"team": "platform"}},
				{ID: "id-2", Name: "acme/web", URL: "https://github.com/acme/web", Provider: "github"},
			}
		case "2":
			page.Page = 2
			page.Repos = []Repo{{ID: "id-3", Name: "acme/worker", URL: "https://gitlab.com/acme/worker", Provider: "gitlab"}}
		default:
			t.Errorf("unexpected page: %q", r.URL.Query().Get("page"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	})

	repos, err := c.ListRepos(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repos) != 3 {
		t.Fatalf("expected 3 repos across pages, got %d", len(repos))
	}
	if repos[0].ID != "id-1" || repos[0].Tags["team"] != "platform" {
		t.Errorf("unexpected first repo: %+v", repos[0])
	}
	if repos[2].ID != "id-3" || repos[2].Provider != "gitlab" {
		t.Errorf("unexpected repo from page 2: %+v", repos[2])
	}
}

func TestListRepos_Empty(t *testing.T) {
	c := newRepoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ReposPage{Page: 1, PerPage: 50, TotalPages: 0, TotalCount: 0, Repos: []Repo{}})
	})

	repos, err := c.ListRepos(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected no repos, got %d", len(repos))
	}
}

func TestListRepos_Filters(t *testing.T) {
	c := newRepoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		want := map[string]string{
			"name":         "acme/api",
			"search":       "",
			"provider":     "github",
			"vcs_instance": "github.com",
			"namespace":    "acme/platform",
		}
		for k, v := range want {
			if got := q.Get(k); got != v {
				t.Errorf("expected %s=%q, got %q", k, v, got)
			}
		}
		if _, ok := q["search"]; ok {
			t.Error("expected empty search filter to be omitted")
		}
		if got := q["tag"]; !reflect.DeepEqual(got, []string{"team", "tier:1"}) {
			t.Errorf("expected repeated tag params [team tier:1], got %v", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ReposPage{Page: 1, PerPage: 50, TotalPages: 1, TotalCount: 0, Repos: []Repo{}})
	})

	_, err := c.ListRepos(context.Background(), &ListReposOptions{
		Name:        "acme/api",
		Provider:    "github",
		VCSInstance: "github.com",
		Namespace:   "acme/platform",
		Tags:        []string{"team", "tier:1"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListRepos_Search(t *testing.T) {
	c := newRepoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("search"); got != "api" {
			t.Errorf("expected search=api, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ReposPage{Page: 1, PerPage: 50, TotalPages: 1, Repos: []Repo{}})
	})

	if _, err := c.ListRepos(context.Background(), &ListReposOptions{Search: "api"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListRepos_Error(t *testing.T) {
	c := newRepoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "name and search are mutually exclusive"}`))
	})

	if _, err := c.ListRepos(context.Background(), nil); err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetRepo_ByInternalID(t *testing.T) {
	c := newRepoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repos/test-org/abc-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("id"); got != "abc-123" {
			t.Errorf("expected id=abc-123, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "abc-123",
			"name": "acme/platform/api",
			"short_name": "api",
			"url": "https://github.com/acme/platform/api",
			"provider": "github",
			"vcs_instance": "github.com",
			"tags": {"team": "platform"}
		}`))
	})

	repo, err := c.GetRepo(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := &Repo{
		ID:          "abc-123",
		Name:        "acme/platform/api",
		URL:         "https://github.com/acme/platform/api",
		Provider:    "github",
		VCSInstance: "github.com",
		Tags:        map[string]string{"team": "platform"},
	}
	if !reflect.DeepEqual(repo, want) {
		t.Errorf("unexpected repo:\n got %+v\nwant %+v", repo, want)
	}
}

func TestGetRepo_NotFound(t *testing.T) {
	c := newRepoTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Repo 'abc-123' not found"}`))
	})

	_, err := c.GetRepo(context.Background(), "abc-123")
	if !IsNotFound(err) {
		t.Fatalf("expected a not-found error, got %v", err)
	}
}
