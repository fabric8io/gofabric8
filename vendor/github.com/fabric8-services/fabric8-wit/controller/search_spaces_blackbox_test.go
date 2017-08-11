package controller_test

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"context"

	"github.com/goadesign/goa"
)

type args struct {
	pageOffset *string
	pageLimit  *int
	q          string
}

type expect func(*testing.T, okScenario, *app.SearchSpaceList)
type expects []expect

type okScenario struct {
	name    string
	args    args
	expects expects
}

type TestSearchSpacesREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunSearchSpacesREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSearchSpacesREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSearchSpacesREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestSearchSpacesREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSearchSpacesREST) SecuredController() (*goa.Service, *SearchController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Search-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewSearchController(svc, rest.db, rest.Configuration)
}

func (rest *TestSearchSpacesREST) UnSecuredController() (*goa.Service, *SearchController) {
	svc := goa.New("Search-Service")
	return svc, NewSearchController(svc, rest.db, rest.Configuration)
}

func (rest *TestSearchSpacesREST) TestSpacesSearchOK() {
	// given
	idents, err := createTestData(rest.db)
	require.Nil(rest.T(), err)
	tests := []okScenario{
		{"With uppercase fullname query", args{offset("0"), limit(10), "TEST_AB"}, expects{totalCount(1)}},
		{"With lowercase fullname query", args{offset("0"), limit(10), "TEST_AB"}, expects{totalCount(1)}},
		{"With uppercase description query", args{offset("0"), limit(10), "DESCRIPTION FOR TEST_AB"}, expects{totalCount(1)}},
		{"With lowercase description query", args{offset("0"), limit(10), "description for test_ab"}, expects{totalCount(1)}},
		{"with special chars", args{offset("0"), limit(10), "&:\n!#%?*"}, expects{totalCount(0)}},
		{"with * to list all", args{offset("0"), limit(10), "*"}, expects{totalCountAtLeast(len(idents))}},
		{"with multi page", args{offset("0"), limit(10), "TEST"}, expects{hasLinks("Next")}},
		{"with last page", args{offset(strconv.Itoa(len(idents) - 1)), limit(10), "TEST"}, expects{hasNoLinks("Next"), hasLinks("Prev")}},
		{"with different values", args{offset("0"), limit(10), "TEST"}, expects{differentValues()}},
	}
	svc, ctrl := rest.UnSecuredController()
	// when/then
	for _, tt := range tests {
		_, result := test.SpacesSearchOK(rest.T(), svc.Context, svc, ctrl, tt.args.pageLimit, tt.args.pageOffset, tt.args.q)
		for _, expect := range tt.expects {
			expect(rest.T(), tt, result)
		}
	}
}

func createTestData(db application.DB) ([]space.Space, error) {
	names := []string{"TEST_A", "TEST_AB", "TEST_B", "TEST_C"}
	for i := 0; i < 20; i++ {
		names = append(names, "TEST_"+strconv.Itoa(i))
	}

	spaces := []space.Space{}

	err := application.Transactional(db, func(app application.Application) error {
		for _, name := range names {
			space := space.Space{
				Name:        name,
				Description: strings.ToTitle("description for " + name),
			}
			newSpace, err := app.Spaces().Create(context.Background(), &space)
			if err != nil {
				return err
			}
			spaces = append(spaces, *newSpace)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to insert testdata %v", err)
	}
	return spaces, nil
}

func totalCount(count int) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		if got := result.Meta.TotalCount; got != count {
			t.Errorf("%s got = %v, want %v", scenario.name, got, count)
		}
	}
}

func totalCountAtLeast(count int) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		got := result.Meta.TotalCount
		if !(got >= count) {
			t.Errorf("%s got %v, wanted at least %v", scenario.name, got, count)
		}
	}
}

func hasLinks(linkNames ...string) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		for _, linkName := range linkNames {
			link := linkName
			if reflect.Indirect(reflect.ValueOf(result.Links)).FieldByName(link).IsNil() {
				t.Errorf("%s got empty link, wanted %s", scenario.name, link)
			}
		}
	}
}

func hasNoLinks(linkNames ...string) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		for _, linkName := range linkNames {
			if !reflect.Indirect(reflect.ValueOf(result.Links)).FieldByName(linkName).IsNil() {
				t.Errorf("%s got link, wanted empty %s", scenario.name, linkName)
			}
		}
	}
}

func differentValues() expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		var prev *app.Space

		for i := range result.Data {
			s := result.Data[i]
			if prev == nil {
				prev = s
			} else {
				if *prev.Attributes.Name == *s.Attributes.Name {
					t.Errorf("%s got equal name, wanted different %s", scenario.name, *s.Attributes.Name)
				}
				if *prev.Attributes.Description == *s.Attributes.Description {
					t.Errorf("%s got equal description, wanted different %s", scenario.name, *s.Attributes.Description)
				}
				if *prev.ID == *s.ID {
					t.Errorf("%s got equal ID, wanted different %s", scenario.name, *s.ID)
				}
				if prev.Type != s.Type {
					t.Errorf("%s got non equal Type, wanted same %s", scenario.name, s.Type)
				}
			}
		}
	}
}

func limit(n int) *int {
	return &n
}
func offset(n string) *string {
	return &n
}
