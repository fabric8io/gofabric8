package remoteworkitem

import (
	"encoding/json"

	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// githubFetcher provides issue listing
type githubFetcher interface {
	listIssues(query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error)
}

// GithubTracker represents the Github tracker provider
type GithubTracker struct {
	URL   string
	Query string
}

// GithubIssueFetcher fetch issues from github
type githubIssueFetcher struct {
	client *github.Client
}

// ListIssues list all issues
func (f *githubIssueFetcher) listIssues(query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error) {
	return f.client.Search.Issues(query, opts)
}

// Fetch tracker items from Github
func (g *GithubTracker) Fetch(githubAuthToken string) chan TrackerItemContent {
	f := githubIssueFetcher{}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAuthToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	f.client = github.NewClient(tc)
	return g.fetch(&f)
}

func (g *GithubTracker) fetch(f githubFetcher) chan TrackerItemContent {
	item := make(chan TrackerItemContent)
	go func() {
		opts := &github.SearchOptions{
			ListOptions: github.ListOptions{
				PerPage: 20,
			},
		}
		for {
			result, response, err := f.listIssues(g.Query, opts)
			if _, ok := err.(*github.RateLimitError); ok {
				log.Warn(nil, map[string]interface{}{
					"query": g.Query,
					"opts":  opts,
				}, "reached rate limit when listing Github issues")
				break
			}
			issues := result.Issues
			for _, l := range issues {
				id, _ := json.Marshal(l.URL)
				content, _ := json.Marshal(l)
				item <- TrackerItemContent{ID: string(id), Content: content}
			}
			if response.NextPage == 0 {
				break
			}
			opts.ListOptions.Page = response.NextPage
		}
		close(item)
	}()
	return item
}
