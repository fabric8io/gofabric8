package cmds

import (
	"testing"

	"github.com/openshift/origin/pkg/project/api"
)

func TestGetProject(t *testing.T) {
	p1 := api.Project{}
	p2 := api.Project{}
	p3 := api.Project{}
	p4 := api.Project{}

	p1.Name = "foo-che"
	p2.Name = "bar-che"
	p3.Name = "foo"
	p4.Name = "moto"
	// Pick the first one if when we have multiples projects and we are
	// currently in an unrelated project
	res := detectCurrentUserProject("moto", []api.Project{p1, p2, p3, p4})
	if res != p3.Name {
		t.Fatalf("%s != foo", res)
	}

	p1.Name = "foo-che"
	p2.Name = "bar-che"
	p3.Name = "foo"
	p4.Name = "bar"
	// Return the second project cause we are currently in there
	res = detectCurrentUserProject("bar-che", []api.Project{p1, p2, p3, p4})
	if res != "bar" {
		t.Fatalf("%s != bar", res)
	}

	p1.Name = "foo-che"
	p2.Name = "bar-che"
	p3.Name = "foo"
	p4.Name = "bar"
	// Return bar because we are currently in the namespace bar-jenkins which has the same
	// prefix)
	res = detectCurrentUserProject("bar-jenkins", []api.Project{p1, p2, p3, p4})
	if res != "bar" {
		t.Fatalf("%s != %s", res, "bar")
	}

	// Return an error here, cause we have a foo-che but we don't have a parent
	// project without prefix (i.e: foo)
	p1.Name = "foo-che"
	p2.Name = "moto"
	res = detectCurrentUserProject("moto", []api.Project{p1, p2})
	if res != "" {
		t.Fatalf("%s != foo", res)
	}

	// test if we can get properly the -jenkins and not just the *-che
	p1.Name = "foo"
	p2.Name = "foo-jenkins"
	p3.Name = "hellomoto"
	res = detectCurrentUserProject("hellomoto", []api.Project{p1, p2, p3})
	if res != "foo" {
		t.Fatalf("%s != foo", res)
	}

}
