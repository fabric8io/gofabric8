package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, 5, len(params), "unknown number of parameters")
}

func TestFoundJenkinsQuotasOSO(t *testing.T) {
	c, err := Asset("template/fabric8-online-jenkins-quotas-oso-openshift.yml")
	if err != nil {
		t.Fatalf("Asset template/fabric8-online-jenkins-quotas-oso-openshift.yml not found")
	}

	cs := string(c)
	if !strings.Contains(cs, "Limit") {
		t.Fatalf("Word Limit not found in the resource")
	}

	var template map[interface{}]interface{}
	err = yaml.Unmarshal(c, &template)
	if err != nil {
		t.Fatalf("Could not parse resource as yaml")
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

	assert.Equal(t, 7, len(params), "unknown number of parameters")
}

func TestFoundCheQuotasOSO(t *testing.T) {
	c, err := Asset("template/fabric8-online-che-quotas-oso-openshift.yml")
	if err != nil {
		t.Fatalf("Asset template/fabric8-online-che-quotas-oso-openshift.yml not found")
	}

	cs := string(c)
	if !strings.Contains(cs, "Limit") {
		t.Fatalf("Word Limit not found in the resource")
	}

	var template map[interface{}]interface{}
	err = yaml.Unmarshal(c, &template)
	if err != nil {
		t.Fatalf("Could not parse resource as yaml")
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
	assert.Equal(t, 5, len(params), "unknown number of parameters")
}
