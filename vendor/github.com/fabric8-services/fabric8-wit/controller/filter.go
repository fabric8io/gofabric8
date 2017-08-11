package controller

import (
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
)

// FilterController implements the filter resource.
type FilterController struct {
	*goa.Controller
	config FilterControllerConfiguration
}

// FilterControllerConfiguration the configuration for the FilterController.
type FilterControllerConfiguration interface {
	GetCacheControlFilters() string
}

// NewFilterController creates a filter controller.
func NewFilterController(service *goa.Service, config FilterControllerConfiguration) *FilterController {
	return &FilterController{
		Controller: service.NewController("FilterController"),
		config:     config,
	}
}

// List runs the list action.
func (c *FilterController) List(ctx *app.ListFilterContext) error {
	var arr []*app.Filters
	arr = append(arr, &app.Filters{
		Attributes: &app.FilterAttributes{
			Title:       "Assignee",
			Query:       "filter[assignee]={id}",
			Description: "Filter by assignee",
			Type:        "users",
		},
		Type: "filters",
	},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Area",
				Query:       "filter[area]={id}",
				Description: "Filter by area",
				Type:        "areas",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Iteration",
				Query:       "filter[iteration]={id}",
				Description: "Filter by iteration",
				Type:        "iterations",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Workitem type",
				Query:       "filter[workitemtype]={id}",
				Description: "Filter by workitemtype",
				Type:        "workitemtypes",
			},
			Type: "filters",
		},
		&app.Filters{
			Attributes: &app.FilterAttributes{
				Title:       "Workitem state",
				Query:       "filter[workitemstate]={id}",
				Description: "Filter by workitemstate",
				Type:        "workitemstate",
			},
			Type: "filters",
		},
	)
	result := &app.FilterList{
		Data: arr,
	}
	// compute an ETag based on the type and query of each filter
	filterEtagData := make([]app.ConditionalRequestEntity, len(result.Data))
	for i, filter := range result.Data {
		filterEtagData[i] = FilterEtagData{
			Type:  filter.Attributes.Type,
			Query: filter.Attributes.Query,
		}
	}
	ctx.ResponseData.Header().Set(app.ETag, app.GenerateEntitiesTag(filterEtagData))
	// set now as the last modified date
	ctx.ResponseData.Header().Set(app.LastModified, app.ToHTTPTime(time.Now()))
	// cache-control
	ctx.ResponseData.Header().Set(app.CacheControl, c.config.GetCacheControlFilters())
	return ctx.OK(result)
}

func addFilterLinks(links *app.PagingLinks, request *goa.RequestData) {
	filter := rest.AbsoluteURL(request, app.FilterHref())
	links.Filters = &filter
}

// FilterEtagData structure that carries the data to generate an ETag.
type FilterEtagData struct {
	Type  string
	Query string
}

// GetETagData returns the field values to compute the ETag.
func (f FilterEtagData) GetETagData() []interface{} {
	return []interface{}{f.Type, f.Query}
}

// GetLastModified returns the field values to compute the '`Last-Modified` response header.
func (f FilterEtagData) GetLastModified() time.Time {
	return time.Now()
}
