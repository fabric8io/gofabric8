package controller_test

import (
	"testing"

	"time"

	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/suite"
)

type TestStatusREST struct {
	gormtestsupport.DBTestSuite

	clean func()
}

func TestRunStatusREST(t *testing.T) {
	suite.Run(t, &TestStatusREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestStatusREST) SetupTest() {
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestStatusREST) TearDownTest() {
	rest.clean()
}

func (rest *TestStatusREST) SecuredController() (*goa.Service, *StatusController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Status-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewStatusController(svc, rest.DB)
}

func (rest *TestStatusREST) UnSecuredController() (*goa.Service, *StatusController) {
	svc := goa.New("Status-Service")
	return svc, NewStatusController(svc, rest.DB)
}

func (rest *TestStatusREST) TestShowStatusOK() {
	t := rest.T()
	resource.Require(t, resource.Database)
	svc, ctrl := rest.UnSecuredController()
	_, res := test.ShowStatusOK(t, svc.Context, svc, ctrl)

	if res.Commit != "0" {
		t.Error("Commit not found")
	}
	if res.StartTime != StartTime {
		t.Error("StartTime is not correct")
	}
	_, err := time.Parse("2006-01-02T15:04:05Z", res.StartTime)
	if err != nil {
		t.Error("Incorrect layout of StartTime: ", err.Error())
	}
}
