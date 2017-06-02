package openshift

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabric8io/fabric8-init-tenant/template"
)

const (
	varProjectName           = "PROJECT_NAME"
	varProjectTemplateName   = "PROJECT_TEMPLATE_NAME"
	varProjectDisplayName    = "PROJECT_DISPLAYNAME"
	varProjectDescription    = "PROJECT_DESCRIPTION"
	varProjectUser           = "PROJECT_USER"
	varProjectRequestingUser = "PROJECT_REQUESTING_USER"
	varProjectAdminUser      = "PROJECT_ADMIN_USER"
	varProjectNamespace      = "PROJECT_NAMESPACE"
)

// InitTenant initializes a new tenant in openshift
// Creates the new x-test|stage|run and x-jenkins|che namespaces
// and install the required services/routes/deployment configurations to run
// e.g. Jenkins and Che
func InitTenant(config Config, callback Callback, username, usertoken string, templateVars map[string]string) error {
	err := do(config, callback, username, usertoken, templateVars)
	if err != nil {
		return err
	}
	return nil
}

func do(config Config, callback Callback, username, usertoken string, templateVars map[string]string) error {
	name := createName(username)

	vars := map[string]string{
		varProjectName:           name,
		varProjectTemplateName:   name,
		varProjectDisplayName:    name,
		varProjectDescription:    name,
		varProjectUser:           username,
		varProjectRequestingUser: username,
		varProjectAdminUser:      config.MasterUser,
	}

	for k, v := range templateVars {
		if _, exist := vars[k]; !exist {
			vars[k] = v
		}
	}

	masterOpts := ApplyOptions{Config: config, Callback: callback}
	userOpts := ApplyOptions{Config: config.WithToken(usertoken), Namespace: name, Callback: callback}

	userProjectT, err := loadTemplate(config, "fabric8-online-user-project.yml")
	if err != nil {
		return err
	}

	userProjectRolesT, err := loadTemplate(config, "fabric8-online-user-rolebindings.yml")
	if err != nil {
		return err
	}

	userProjectCollabT, err := loadTemplate(config, "fabric8-online-user-colaborators.yml")
	if err != nil {
		return err
	}

	projectT, err := loadTemplate(config, "fabric8-online-team-openshift.yml")
	if err != nil {
		return err
	}

	jenkinsT, err := loadTemplate(config, "fabric8-online-jenkins-openshift.yml")
	if err != nil {
		return err
	}
	cheT, err := loadTemplate(config, "fabric8-online-che-openshift.yml")
	if err != nil {
		return err
	}

	var channels []chan error

	err = executeNamespaceSync(string(userProjectT), vars, userOpts)
	if err != nil {
		return err
	}

	err = executeNamespaceSync(string(userProjectCollabT), vars, masterOpts.WithNamespace(name))
	if err != nil {
		return err
	}

	err = executeNamespaceSync(string(userProjectRolesT), vars, userOpts.WithNamespace(name))
	if err != nil {
		return err
	}

	{
		lvars := clone(vars)
		lvars[varProjectDisplayName] = lvars[varProjectName]

		err = executeNamespaceSync(string(projectT), lvars, masterOpts.WithNamespace(name))
		if err != nil {
			return err
		}
	}

	{
		lvars := clone(vars)
		nsname := fmt.Sprintf("%v-jenkins", name)
		lvars[varProjectNamespace] = vars[varProjectName]
		ns := executeNamespaceAsync(string(jenkinsT), lvars, masterOpts.WithNamespace(nsname))
		channels = append(channels, ns)
	}
	{
		lvars := clone(vars)
		nsname := fmt.Sprintf("%v-che", name)
		lvars[varProjectNamespace] = vars[varProjectName]
		ns := executeNamespaceAsync(string(cheT), lvars, masterOpts.WithNamespace(nsname))
		channels = append(channels, ns)
	}

	var errors []error
	for _, channel := range channels {
		err := <-channel
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return multiError{Errors: errors}
	}
	return nil
}

// loadTemplate will load the template for a specific version from maven central or from the template directory
// or default to the OOTB template included
func loadTemplate(config Config, name string) ([]byte, error) {
	teamVersion := config.TeamVersion
	logCallback := config.GetLogCallback()
	if len(teamVersion) > 0 {
		url := ""
		switch name {
		case "fabric8-online-team-openshift.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-team/$TEAM_VERSION/fabric8-online-team-$TEAM_VERSION-openshift.yml"
		case "fabric8-online-jenkins-openshift.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-jenkins/$TEAM_VERSION/fabric8-online-jenkins-$TEAM_VERSION-openshift.yml"
		case "fabric8-online-che-openshift.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-che/$TEAM_VERSION/fabric8-online-che-$TEAM_VERSION-openshift.yml"
		}
		if len(url) > 0 {
			url = strings.Replace(url, "$TEAM_VERSION", teamVersion, -1)
			logCallback(fmt.Sprintf("Loading template from URL: %s", url))
			resp, err := http.Get(url)
			if err != nil {
				return nil, fmt.Errorf("Failed to load template from %s due to: %v", url, err)
			}
			defer resp.Body.Close()
			statusCode := resp.StatusCode
			if statusCode >= 300 {
				return nil, fmt.Errorf("Failed to GET template from %s got status code to: %d", url, statusCode)
			}
			return ioutil.ReadAll(resp.Body)
		}
	}
	dir := config.TemplateDir
	if len(dir) > 0 {
		fullName := filepath.Join(dir, name)
		d, err := os.Stat(fullName)
		if err == nil {
			if m := d.Mode(); m.IsRegular() {
				logCallback(fmt.Sprintf("Loading template from file: %s", fullName))
				return ioutil.ReadFile(fullName)
			}
		}
	}
	return template.Asset("template/" + name)
}

func createName(username string) string {
	return strings.Replace(strings.Split(username, "@")[0], ".", "-", -1)
}

func executeNamespaceSync(template string, vars map[string]string, opts ApplyOptions) error {
	t, err := Process(template, vars)
	if err != nil {
		return err
	}
	err = Apply(t, opts)
	if err != nil {
		return err
	}
	return nil
}

func executeNamespaceAsync(template string, vars map[string]string, opts ApplyOptions) chan error {
	ch := make(chan error)
	go func() {
		t, err := Process(template, vars)
		if err != nil {
			ch <- err
		}

		err = Apply(t, opts)
		if err != nil {
			ch <- err
		}

		ch <- nil
		close(ch)
	}()
	return ch
}

func clone(maps map[string]string) map[string]string {
	maps2 := make(map[string]string)
	for k2, v2 := range maps {
		maps2[k2] = v2
	}
	return maps2
}
