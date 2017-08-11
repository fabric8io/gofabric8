package controller

import (
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
)

var (
	// Commit current build commit set by build script
	Commit = "0"
	// BuildTime set by build script in ISO 8601 (UTC) format: YYYY-MM-DDThh:mm:ssTZD (see https://www.w3.org/TR/NOTE-datetime for details)
	BuildTime = "0"
	// StartTime in ISO 8601 (UTC) format
	StartTime = time.Now().UTC().Format("2006-01-02T15:04:05Z")
)

// StatusController implements the status resource.
type StatusController struct {
	*goa.Controller
	db *gorm.DB
}

// NewStatusController creates a status controller.
func NewStatusController(service *goa.Service, db *gorm.DB) *StatusController {
	return &StatusController{
		Controller: service.NewController("StatusController"),
		db:         db,
	}
}

// Show runs the show action.
func (c *StatusController) Show(ctx *app.ShowStatusContext) error {
	res := &app.Status{}
	res.Commit = Commit
	res.BuildTime = BuildTime
	res.StartTime = StartTime

	_, err := c.db.DB().Exec("select 1")
	if err != nil {
		var message string
		message = err.Error()
		res.Error = &message
		return ctx.ServiceUnavailable(res)
	}
	return ctx.OK(res)
}
