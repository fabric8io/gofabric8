package remoteworkitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestLookupProvider(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	ts1 := trackerSchedule{TrackerType: ProviderGithub}
	tp1 := lookupProvider(ts1)
	require.NotNil(t, tp1)

	ts2 := trackerSchedule{TrackerType: ProviderJira}
	tp2 := lookupProvider(ts2)
	require.NotNil(t, tp2)

	ts3 := trackerSchedule{TrackerType: "unknown"}
	tp3 := lookupProvider(ts3)
	require.Nil(t, tp3)
}
