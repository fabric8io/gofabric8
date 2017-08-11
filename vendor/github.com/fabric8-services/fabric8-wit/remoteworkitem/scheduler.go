package remoteworkitem

import (
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/models"

	"context"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	uuid "github.com/satori/go.uuid"
)

// TrackerSchedule capture all configuration
type trackerSchedule struct {
	TrackerID   int
	URL         string
	TrackerType string
	Query       string
	Schedule    string
	SpaceID     uuid.UUID
}

// Scheduler represents scheduler
type Scheduler struct {
	db *gorm.DB
}

var cr *cron.Cron

// NewScheduler creates a new Scheduler
func NewScheduler(db *gorm.DB) *Scheduler {
	s := Scheduler{db: db}
	return &s
}

// Stop scheduler
// This should be called only from main
func (s *Scheduler) Stop() {
	cr.Stop()
}

func batchID() string {
	u1 := uuid.NewV4().String()
	return u1
}

// ScheduleAllQueries fetch and import of remote tracker items
func (s *Scheduler) ScheduleAllQueries(ctx context.Context, accessTokens map[string]string) {
	cr.Stop()

	trackerQueries := fetchTrackerQueries(s.db)
	for _, tq := range trackerQueries {
		cr.AddFunc(tq.Schedule, func() {
			tr := lookupProvider(tq)
			authToken := accessTokens[tq.TrackerType]

			// In case of Jira, no auth token is needed hence the map wouldnt
			// return anything. So effectively the authToken is optional.

			for i := range tr.Fetch(authToken) {
				models.Transactional(s.db, func(tx *gorm.DB) error {
					// Save the remote items in a 'temporary' table.
					err := upload(tx, tq.TrackerID, i)
					if err != nil {
						return errors.WithStack(err)
					}
					// Convert the remote item into a local work item and persist in the DB.
					_, err = convertToWorkItemModel(ctx, tx, tq.TrackerID, i, tq.TrackerType, tq.SpaceID)
					return errors.WithStack(err)
				})
			}
		})
	}
	cr.Start()
}

func fetchTrackerQueries(db *gorm.DB) []trackerSchedule {
	tsList := []trackerSchedule{}
	err := db.Table("tracker_queries").Select("trackers.id as tracker_id, trackers.url, trackers.type as tracker_type, tracker_queries.query, tracker_queries.schedule, tracker_queries.space_id").Joins("left join trackers on tracker_queries.tracker_id = trackers.id").Where("trackers.deleted_at is NULL AND tracker_queries.deleted_at is NULL").Scan(&tsList).Error
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
		}, "fetch operation failed for tracker queries")
	}
	return tsList
}

// lookupProvider provides the respective tracker based on the type
func lookupProvider(ts trackerSchedule) TrackerProvider {
	switch ts.TrackerType {
	case ProviderGithub:
		return &GithubTracker{URL: ts.URL, Query: ts.Query}
	case ProviderJira:
		return &JiraTracker{URL: ts.URL, Query: ts.Query}
	}
	return nil
}

// TrackerItemContent represents a remote tracker item with it's content and unique ID
type TrackerItemContent struct {
	ID      string
	Content []byte
}

// TrackerProvider represents a remote tracker
type TrackerProvider interface {
	Fetch(authToken string) chan TrackerItemContent // TODO: Change to an interface to enforce the contract
}

func init() {
	cr = cron.New()
	cr.Start()
}
