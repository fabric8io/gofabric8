package migration

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	config "github.com/fabric8-services/fabric8-tenant/configuration"
	"github.com/fabric8-services/fabric8-wit/resource"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentMigrations(t *testing.T) {
	resource.Require(t, resource.Database)

	configuration, err := config.NewData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			db, err := sql.Open("postgres", configuration.GetPostgresConfigString())
			if err != nil {
				t.Fatalf("Cannot connect to DB: %s\n", err)
			}
			err = Migrate(db)
			assert.Nil(t, err)
		}()

	}
	wg.Wait()
}
