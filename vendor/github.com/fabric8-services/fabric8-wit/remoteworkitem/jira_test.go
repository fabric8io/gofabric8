package remoteworkitem

import (
	"net/http"
	"testing"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeJiraIssueFetcher struct{}

func (f *fakeJiraIssueFetcher) listIssues(jql string, options *jira.SearchOptions) ([]jira.Issue, *jira.Response, error) {
	return []jira.Issue{{}}, &jira.Response{}, nil
}

func (f *fakeJiraIssueFetcher) getIssue(issueID string) (*jira.Issue, *jira.Response, error) {
	return &jira.Issue{ID: "1"}, &jira.Response{}, nil
}

func TestJiraFetch(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	f := fakeJiraIssueFetcher{}
	j := JiraTracker{URL: "", Query: ""}
	// when
	i := <-j.fetch(&f)
	// then
	assert.Equal(t, `{"id":"1"}`, string(i.Content))
}

func TestJiraFetchWithRecording(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	recorder, err := recorder.New("../test/data/jira_fetch_test")
	require.Nil(t, err)
	defer recorder.Stop()
	h := &http.Client{
		Timeout:   100 * time.Second,
		Transport: recorder.Transport,
	}
	f := jiraIssueFetcher{}
	j := JiraTracker{URL: "https://issues.jboss.org", Query: "project = Arquillian AND status = Closed AND assignee = aslak AND fixVersion = 1.1.11.Final AND priority = Major ORDER BY created ASC"}
	client, err := jira.NewClient(h, j.URL)
	require.Nil(t, err)
	f.client = client
	// when
	trackerItemContentChannel := j.fetch(&f)
	// then
	require.NotNil(t, trackerItemContentChannel)
	// collect items from channel and store in slice
	var trackerItemContents []TrackerItemContent
	for trackerItemContent := range trackerItemContentChannel {
		trackerItemContents = append(trackerItemContents, trackerItemContent)
	}
	require.Len(t, trackerItemContents, 5, "Retrieved tracker item contents")
	assert.Equal(t, `"ARQ-1937"`, trackerItemContents[0].ID)
	assert.Equal(t, `"ARQ-1956"`, trackerItemContents[1].ID)
	assert.Equal(t, `"ARQ-1996"`, trackerItemContents[2].ID)
	assert.Equal(t, `"ARQ-2009"`, trackerItemContents[3].ID)
	assert.Equal(t, `"ARQ-2010"`, trackerItemContents[4].ID)
}
