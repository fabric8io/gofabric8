package openshift

import (
	"fmt"
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

	userProjectT, err := template.Asset("template/fabric8-online-user-project.yml")
	if err != nil {
		return err
	}

	userProjectRolesT, err := template.Asset("template/fabric8-online-user-rolebindings.yml")
	if err != nil {
		return err
	}

	userProjectCollabT, err := template.Asset("template/fabric8-online-user-colaborators.yml")
	if err != nil {
		return err
	}

	projectT, err := template.Asset("template/fabric8-online-team-openshift.yml")
	if err != nil {
		return err
	}

	jenkinsT, err := template.Asset("template/fabric8-online-jenkins-openshift.yml")
	if err != nil {
		return err
	}
	cheT, err := template.Asset("template/fabric8-online-che-openshift.yml")
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
