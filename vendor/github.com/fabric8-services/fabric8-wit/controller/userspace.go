package controller

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/models"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// Data describes a single resource in usersapce
type Data struct {
	gormsupport.Lifecycle
	ID   uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	Path string
	Data workitem.Fields `sql:"type:jsonb"`
}

// TableName implements gorm.tabler
func (w Data) TableName() string {
	return "userspace_data"
}

// UserspaceController implements the userspace resource.
type UserspaceController struct {
	*goa.Controller
	db *gorm.DB
}

// NewUserspaceController creates a userspace controller.
func NewUserspaceController(service *goa.Service, db *gorm.DB) *UserspaceController {
	db.AutoMigrate(&Data{}).AddUniqueIndex("idx_userspace_path", "path")

	return &UserspaceController{Controller: service.NewController("UserspaceController"), db: db}
}

// Create runs the create action.
func (c *UserspaceController) Create(ctx *app.CreateUserspaceContext) error {
	return models.Transactional(c.db, func(db *gorm.DB) error {

		path := ctx.RequestURI

		data := Data{}
		err := c.db.Where("path = ?", path).First(&data).Error
		fmt.Println(err)
		if err != nil {
			data = Data{
				ID:   uuid.NewV4(),
				Path: ctx.RequestURI,
				Data: workitem.Fields(ctx.Payload),
			}
			err := c.db.Create(&data).Error
			if err != nil {
				goa.LogError(ctx, "error adding data", "error", err.Error())
				return ctx.InternalServerError()
			}
		} else {
			err := c.db.Model(&data).Update("data", workitem.Fields(ctx.Payload)).Error
			if err != nil {
				goa.LogError(ctx, "error updating data", "error", err.Error())
				return ctx.InternalServerError()
			}
		}
		return ctx.NoContent()
	})
}

// Show shows the record
func (c *UserspaceController) Show(ctx *app.ShowUserspaceContext) error {
	return models.Transactional(c.db, func(db *gorm.DB) error {

		path := ctx.RequestURI
		data := Data{}
		err := c.db.Where("path = ?", path).First(&data).Error
		if err != nil {
			return ctx.NotFound()
		}

		return ctx.OK(data.Data)
	})
}
