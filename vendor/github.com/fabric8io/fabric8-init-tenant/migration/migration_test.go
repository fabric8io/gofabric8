package migration

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/almighty/almighty-core/resource"
	config "github.com/fabric8io/fabric8-init-tenant/configuration"

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
