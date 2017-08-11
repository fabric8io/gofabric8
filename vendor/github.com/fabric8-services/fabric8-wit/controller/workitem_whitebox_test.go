package controller

import (
	"context"
	"os"
	"testing"

	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestParseInts(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	integers, err := parseInts(nil)
	assert.Equal(t, nil, err)
	assert.Equal(t, []int{}, integers)

	str := "1, 2, foo"
	_, err = parseInts(&str)
	assert.NotNil(t, err)
}

func TestParseLimit(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Test parsing error in parseInts
	str := "1000, foo"
	integers, length, err := parseLimit(&str)
	assert.NotNil(t, err)
	assert.Equal(t, 0, length)
	assert.Nil(t, integers)

	// Test length = 1
	str = "1000"
	integers, length, err = parseLimit(&str)
	assert.Nil(t, err)
	assert.Equal(t, 1000, length)
	assert.Nil(t, integers)

	// Test empty string
	str = ""
	integers, length, err = parseLimit(&str)
	assert.Nil(t, err)
	assert.Equal(t, 100, length)
	assert.Nil(t, integers)
}

func TestSetPagingLinks(t *testing.T) {
	links := &app.PagingLinks{}
	setPagingLinks(links, "", 0, 0, 1, 0)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.Last)
	assert.Nil(t, links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "prefix", 0, 0, 1, 0)
	assert.Equal(t, "prefix?page[offset]=0&page[limit]=1", *links.First)
	assert.Equal(t, "prefix?page[offset]=0&page[limit]=0", *links.Last)
	assert.Nil(t, links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 0, 1, 1)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 1, 1, 0)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.Last)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Next)
	assert.Nil(t, links.Prev)

	setPagingLinks(links, "", 0, 1, 1, 1)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Next)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Prev)

	setPagingLinks(links, "", 0, 2, 1, 1)
	assert.Equal(t, "?page[offset]=0&page[limit]=0", *links.First)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Last)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Next)
	assert.Equal(t, "?page[offset]=0&page[limit]=1", *links.Prev)

	setPagingLinks(links, "", 0, 3, 4, 4)
	assert.Equal(t, "?page[offset]=0&page[limit]=3", *links.First)
	assert.Equal(t, "?page[offset]=3&page[limit]=4", *links.Last)
	assert.Equal(t, "?page[offset]=3&page[limit]=4", *links.Next)
	assert.Equal(t, "?page[offset]=0&page[limit]=3", *links.Prev)
}

func TestConvertWorkItemWithDescription(t *testing.T) {
	request := http.Request{Host: "localhost"}
	requestData := &goa.RequestData{Request: &request}
	// map[string]interface{}
	fields := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: "description",
	}

	wi := workitem.WorkItem{
		Fields:  fields,
		SpaceID: space.SystemSpace,
	}
	wi2 := ConvertWorkItem(requestData, wi)
	assert.Equal(t, "title", wi2.Attributes[workitem.SystemTitle])
	assert.Equal(t, "description", wi2.Attributes[workitem.SystemDescription])
}

func TestConvertWorkItemWithoutDescription(t *testing.T) {
	request := http.Request{Host: "localhost"}
	requestData := &goa.RequestData{Request: &request}
	wi := workitem.WorkItem{
		Fields: map[string]interface{}{
			workitem.SystemTitle: "title",
		},
		SpaceID: space.SystemSpace,
	}
	wi2 := ConvertWorkItem(requestData, wi)
	assert.Equal(t, "title", wi2.Attributes[workitem.SystemTitle])
	assert.Nil(t, wi2.Attributes[workitem.SystemDescription])
}

type TestWorkItemREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunWorkItemREST(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	suite.Run(t, &TestWorkItemREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestWorkItemREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestWorkItemREST) TearDownTest() {
	rest.clean()
}

func prepareWI2(attributes map[string]interface{}) app.WorkItem {
	spaceRelatedURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	witRelatedURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.WorkitemtypeHref(space.SystemSpace.String(), workitem.SystemBug.String()))
	return app.WorkItem{
		Type: "workitems",
		Relationships: &app.WorkItemRelationships{
			BaseType: &app.RelationBaseType{
				Data: &app.BaseTypeData{
					Type: "workitemtypes",
					ID:   workitem.SystemBug,
				},
				Links: &app.GenericLinks{
					Self:    &witRelatedURL,
					Related: &witRelatedURL,
				},
			},
			Space: app.NewSpaceRelation(space.SystemSpace, spaceRelatedURL),
		},
		Attributes: attributes,
	}
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithLegacyDescription() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: "description",
	}
	source := prepareWI2(attributes)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}

	err := application.Transactional(rest.db, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, space.SystemSpace)
	})
	// assert
	require.Nil(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContentFromLegacy("description")
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])

}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithDescriptionContentNoMarkup() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("description"),
	}
	source := prepareWI2(attributes)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	err := application.Transactional(rest.db, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, space.SystemSpace)
	})
	require.Nil(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContentFromLegacy("description")
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithDescriptionContentAndMarkup() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	attributes := map[string]interface{}{
		workitem.SystemTitle:       "title",
		workitem.SystemDescription: rendering.NewMarkupContent("description", rendering.SystemMarkupMarkdown),
	}
	source := prepareWI2(attributes)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	err := application.Transactional(rest.db, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, space.SystemSpace)
	})
	require.Nil(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	expectedDescription := rendering.NewMarkupContent("description", rendering.SystemMarkupMarkdown)
	assert.Equal(t, expectedDescription, target.Fields[workitem.SystemDescription])
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithTitle() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	title := "title"
	attributes := map[string]interface{}{
		workitem.SystemTitle: title,
	}
	source := prepareWI2(attributes)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	err := application.Transactional(rest.db, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, space.SystemSpace)
	})
	require.Nil(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.Fields)
	require.True(t, uuid.Equal(source.Relationships.BaseType.Data.ID, target.Type))
	assert.Equal(t, title, target.Fields[workitem.SystemTitle])
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithMissingTitle() {
	t := rest.T()
	resource.Require(t, resource.Database)
	//given
	// given
	attributes := map[string]interface{}{}
	source := prepareWI2(attributes)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	// when
	err := application.Transactional(rest.db, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, space.SystemSpace)
	})
	// then: no error expected at this level, even though the title is missing
	require.Nil(t, err)
}

func (rest *TestWorkItemREST) TestConvertJSONAPIToWorkItemWithEmptyTitle() {
	t := rest.T()
	resource.Require(t, resource.Database)
	// given
	attributes := map[string]interface{}{
		workitem.SystemTitle: "",
	}
	source := prepareWI2(attributes)
	target := &workitem.WorkItem{Fields: map[string]interface{}{}}
	// when
	err := application.Transactional(rest.db, func(app application.Application) error {
		return ConvertJSONAPIToWorkItem(context.Background(), "", app, source, target, space.SystemSpace)
	})
	// then: no error expected at this level, even though the title is missing
	require.Nil(t, err)
}
