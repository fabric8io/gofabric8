package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func TestWorkItemLinkCategory_ConvertLinkCategoryFromModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	m := link.WorkItemLinkCategory{
		ID:          uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:        "Example work item link category",
		Description: &description,
		Version:     0,
	}

	expected := app.WorkItemLinkCategorySingle{
		Data: &app.WorkItemLinkCategoryData{
			Type: link.EndpointWorkItemLinkCategories,
			ID:   &m.ID,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &m.Name,
				Description: m.Description,
				Version:     &m.Version,
			},
		},
	}

	actual := ConvertLinkCategoryFromModel(m)
	require.Equal(t, expected.Data.Type, actual.Data.Type)
	require.Equal(t, *expected.Data.ID, *actual.Data.ID)
	require.Equal(t, *expected.Data.Attributes.Name, *actual.Data.Attributes.Name)
	require.Equal(t, *expected.Data.Attributes.Description, *actual.Data.Attributes.Description)
	require.Equal(t, *expected.Data.Attributes.Version, *actual.Data.Attributes.Version)
}
