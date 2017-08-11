package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateIterationsWithCounts(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	spaceID := uuid.NewV4()
	i1 := createMinimumIteration("Spting 1234", spaceID)
	i2 := createMinimumIteration("Spting 1234", spaceID)
	i3 := createMinimumIteration("Spting 1234", spaceID)
	var iterationSlice []*iteration.Iteration
	iterationSlice = append(iterationSlice, i1, i2, i3)
	counts := make(map[string]workitem.WICountsPerIteration)
	counts[i1.ID.String()] = workitem.WICountsPerIteration{
		IterationID: i1.ID.String(),
		Total:       10,
		Closed:      8,
	}
	counts[i2.ID.String()] = workitem.WICountsPerIteration{
		IterationID: i2.ID.String(),
		Total:       3,
		Closed:      1,
	}

	iterationSliceWithCounts := updateIterationsWithCounts(counts)

	for _, iteration := range iterationSlice {
		appIteration := &app.Iteration{
			ID: &iteration.ID,
		}
		iterationSliceWithCounts(nil, iteration, appIteration)
		require.NotNil(t, appIteration.Relationships)
		require.NotNil(t, appIteration.Relationships.Workitems)
		require.NotNil(t, appIteration.Relationships.Workitems.Meta)
		if appIteration.ID.String() == i1.ID.String() {
			assert.Equal(t, 10, appIteration.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 8, appIteration.Relationships.Workitems.Meta["closed"])
		}
		if appIteration.ID.String() == i2.ID.String() {
			assert.Equal(t, 3, appIteration.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 1, appIteration.Relationships.Workitems.Meta["closed"])
		}
		if appIteration.ID.String() == i3.ID.String() {
			assert.Equal(t, 0, appIteration.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 0, appIteration.Relationships.Workitems.Meta["closed"])
		}
	}
}

// helper function to get random iteration.Iteration
func createMinimumIteration(name string, spaceID uuid.UUID) *iteration.Iteration {
	iterationID := uuid.NewV4()
	i := iteration.Iteration{
		ID:      iterationID,
		Name:    name,
		State:   iteration.IterationStateNew,
		SpaceID: spaceID,
	}
	return &i
}
