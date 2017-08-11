package application

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"context"

	uuid "github.com/satori/go.uuid"
)

// TrackerRepository encapsulate storage & retrieval of tracker configuration
type TrackerRepository interface {
	repository.Exister
	Load(ctx context.Context, ID string) (*app.Tracker, error)
	Save(ctx context.Context, t app.Tracker) (*app.Tracker, error)
	Delete(ctx context.Context, ID string) error
	Create(ctx context.Context, url string, typeID string) (*app.Tracker, error)
	List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.Tracker, error)
}

// TrackerQueryRepository encapsulate storage & retrieval of tracker queries
type TrackerQueryRepository interface {
	repository.Exister
	Create(ctx context.Context, query string, schedule string, tracker string, spaceID uuid.UUID) (*app.TrackerQuery, error)
	Save(ctx context.Context, tq app.TrackerQuery) (*app.TrackerQuery, error)
	Load(ctx context.Context, ID string) (*app.TrackerQuery, error)
	Delete(ctx context.Context, ID string) error
	List(ctx context.Context) ([]*app.TrackerQuery, error)
}

// SearchRepository encapsulates searching of woritems,users,etc
type SearchRepository interface {
	SearchFullText(ctx context.Context, searchStr string, start *int, length *int, spaceID *string) ([]workitem.WorkItem, uint64, error)
	Filter(ctx context.Context, filterStr string, start *int, length *int) ([]workitem.WorkItem, uint64, error)
}
