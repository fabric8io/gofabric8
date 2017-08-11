package remoteworkitem

import (
	"net/http"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/recorder"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGithubIssueFetcher struct{}

// ListIssues list all issues
func (f *fakeGithubIssueFetcher) listIssues(query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error) {
	if opts.ListOptions.Page == 0 {
		one := 1
		i := github.Issue{ID: &one}
		isr := &github.IssuesSearchResult{Issues: []github.Issue{i}}
		r := &github.Response{}
		r.NextPage = 1
		return isr, r, nil
	}
	isr := &github.IssuesSearchResult{}
	r := &github.Response{}
	r.NextPage = 0
	return isr, r, nil

}

func TestGithubFetch(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	f := fakeGithubIssueFetcher{}
	g := GithubTracker{URL: "", Query: ""}
	fetch := g.fetch(&f)
	i := <-fetch
	if string(i.Content) != `{"id":1}` {
		t.Errorf("Content is not matching: %#v", string(i.Content))
	}

}

type fakeGithubIssueFetcherWithRateLimit struct{}

// ListIssues list all issues
func (f *fakeGithubIssueFetcherWithRateLimit) listIssues(query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error) {
	isr := &github.IssuesSearchResult{}
	r := &github.Response{}
	r.NextPage = 0
	e := &github.RateLimitError{Message: "rate limit"}
	return isr, r, e
}

func TestGithubFetchWithRateLimit(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	f := fakeGithubIssueFetcherWithRateLimit{}
	g := GithubTracker{URL: "", Query: ""}
	// when
	fetch := g.fetch(&f)
	// then
	assert.Equal(t, 0, len(fetch))
}

func TestGithubFetchWithRecording(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	r, err := recorder.New("../test/data/github_fetch_test")
	require.Nil(t, err)
	defer r.Stop()
	h := &http.Client{
		Timeout:   1 * time.Second,
		Transport: r.Transport,
	}
	f := githubIssueFetcher{}
	f.client = github.NewClient(h)
	g := &GithubTracker{URL: "", Query: "is:open is:issue user:almighty-test"}
	// when
	fetch := g.fetch(&f)
	// then
	i := <-fetch
	assert.Contains(t, string(i.Content), `"html_url":"https://github.com/fabric8-wit-test/fabric8-wit-test-unit/issues/2"`)
	assert.Contains(t, string(i.Content), `"body":"desc\n"`)
	i2 := <-fetch
	assert.Contains(t, string(i2.Content), `"html_url":"https://github.com/fabric8-wit-test/fabric8-wit-test-unit/issues/1"`)
	assert.Contains(t, string(i2.Content), `"body":"sample desc\n"`)
}
