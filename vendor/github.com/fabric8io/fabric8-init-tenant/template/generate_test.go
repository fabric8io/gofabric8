package template

import (
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestFoundJenkins(t *testing.T) {
	c, err := Asset("template/fabric8-online-jenkins-openshift.yml")
	if err != nil {
		t.Fatalf("Asset template/fabric8-online-jenkins-openshift.yml not found")
	}

	cs := string(c)
	if !strings.Contains(cs, "jenkins") {
		t.Fatalf("Word jenkins not found in the template")
	}

	var template map[interface{}]interface{}
	err = yaml.Unmarshal(c, &template)
	if err != nil {
		t.Fatalf("Could not parse template as yaml")
	}

	params, ok := template["parameters"].([]interface{})
	if !ok {
		t.Fatalf("parameters not found")
	}
	if len(params) != 4 {
		t.Fatalf("unknown number of parameters. found %v expected 4", len(params))
	}
}

func TestFoundChe(t *testing.T) {
	c, err := Asset("template/fabric8-online-che-openshift.yml")
	if err != nil {
		t.Fatalf("Asset template/fabric8-online-che-openshift.yml not found")
	}

	cs := string(c)
	if !strings.Contains(cs, "che") {
		t.Fatalf("Word che not found in the template")
	}

	var template map[interface{}]interface{}
	err = yaml.Unmarshal(c, &template)
	if err != nil {
		t.Fatalf("Could not parse template as yaml")
	}

	params, ok := template["parameters"].([]interface{})
	if !ok {
		t.Fatalf("parameters not found")
	}
	if len(params) != 5 {
		t.Fatalf("unknown number of parameters. found %v expected 5", len(params))
	}
}

func TestFoundTeam(t *testing.T) {
	c, err := Asset("template/fabric8-online-team-openshift.yml")
	if err != nil {
		t.Fatalf("Asset template/fabric8-online-team-openshift.yml not found")
	}

	cs := string(c)
	if !strings.Contains(cs, "team") {
		t.Fatalf("Word team not found in the template")
	}

	var template map[interface{}]interface{}
	err = yaml.Unmarshal(c, &template)
	if err != nil {
		t.Fatalf("Could not parse template as yaml")
	}

	params, ok := template["parameters"].([]interface{})
	if !ok {
		t.Fatalf("parameters not found")
	}
	if len(params) != 5 {
		t.Fatalf("unknown number of parameters. found %v expected 5", len(params))
	}
}
