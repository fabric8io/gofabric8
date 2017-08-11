package remoteworkitem

import (
	"encoding/json"

	jira "github.com/andygrunwald/go-jira"
)

// JiraTracker represents the Jira tracker provider
type JiraTracker struct {
	URL   string
	Query string
}

type jiraFetcher interface {
	listIssues(jql string, options *jira.SearchOptions) ([]jira.Issue, *jira.Response, error)
	getIssue(issueID string) (*jira.Issue, *jira.Response, error)
}

type jiraIssueFetcher struct {
	client *jira.Client
}

func (f *jiraIssueFetcher) listIssues(jql string, options *jira.SearchOptions) ([]jira.Issue, *jira.Response, error) {
	return f.client.Issue.Search(jql, options)
}

func (f *jiraIssueFetcher) getIssue(issueID string) (*jira.Issue, *jira.Response, error) {
	return f.client.Issue.Get(issueID)
}

// Fetch collects data from Jira
func (j *JiraTracker) Fetch(authToken string) chan TrackerItemContent {
	f := jiraIssueFetcher{}
	client, _ := jira.NewClient(nil, j.URL)
	f.client = client
	return j.fetch(&f)
}

func (j *JiraTracker) fetch(f jiraFetcher) chan TrackerItemContent {
	item := make(chan TrackerItemContent)
	go func() {
		issues, _, _ := f.listIssues(j.Query, nil)
		for _, l := range issues {
			id, _ := json.Marshal(l.Key)
			issue, _, _ := f.getIssue(l.Key)
			content, _ := json.Marshal(issue)
			item <- TrackerItemContent{ID: string(id), Content: content}
		}
		close(item)
	}()
	return item
}
