package openshift

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sync"

	"github.com/fabric8-services/fabric8-tenant/keycloak"
	"github.com/fabric8-services/fabric8-tenant/template"
	"github.com/fabric8-services/fabric8-tenant/tenant"
	"github.com/fabric8-services/fabric8-wit/log"
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
	varKeycloakURL           = "KEYCLOAK_URL"
)

// InitTenant initializes a new tenant in openshift
// Creates the new x-test|stage|run and x-jenkins|che namespaces
// and install the required services/routes/deployment configurations to run
// e.g. Jenkins and Che
func InitTenant(ctx context.Context, kcConfig keycloak.Config, config Config, callback Callback, username, usertoken string, templateVars map[string]string) error {
	err := do(ctx, kcConfig, config, callback, username, usertoken, templateVars, false)
	if err != nil {
		return err
	}
	return nil
}

//  kcConfig keycloak.Config only used to match signature
func RawInitTenant(ctx context.Context, kcConfig keycloak.Config, config Config, callback Callback, username, usertoken string, templateVars map[string]string) error {
	templs, err := LoadProcessedTemplates(ctx, config, username, templateVars)
	if err != nil {
		return err
	}

	mapped, err := MapByNamespaceAndSort(templs)
	if err != nil {
		return err
	}
	masterOpts := ApplyOptions{Config: config, Callback: callback}
	userOpts := ApplyOptions{Config: config.WithToken(usertoken), Callback: callback}
	var wg sync.WaitGroup
	wg.Add(len(mapped))
	for key, val := range mapped {
		namespaceType := tenant.GetNamespaceType(key)
		if namespaceType == tenant.TypeUser {
			go func(namespace string, objects []map[interface{}]interface{}, opts, userOpts ApplyOptions) {
				defer wg.Done()
				err := ApplyProcessed(Filter(objects, IsOfKind(ValKindProjectRequest, ValKindNamespace)), userOpts)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"namespace": namespace,
						"err":       err,
					}, "error init user project, ProjectRequest")
				}
				err = ApplyProcessed(Filter(objects, IsOfKind(ValKindRoleBindingRestriction)), opts)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"namespace": namespace,
						"err":       err,
					}, "error init user project, RoleBindingRestrictions")
				}
				err = ApplyProcessed(Filter(objects, IsNotOfKind(ValKindProjectRequest, ValKindNamespace, ValKindRoleBindingRestriction)), userOpts)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"namespace": namespace,
						"err":       err,
					}, "error init user project, Other")
				}
			}(key, val, masterOpts, userOpts)
		} else {
			go func(namespace string, objects []map[interface{}]interface{}, opts ApplyOptions) {
				defer wg.Done()
				err := ApplyProcessed(objects, opts)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"namespace": namespace,
						"err":       err,
					}, "error dsaas project")
				}
			}(key, val, masterOpts)
		}
	}
	wg.Wait()
	return nil
}

func UpdateTenant(ctx context.Context, kcConfig keycloak.Config, config Config, callback Callback, username, usertoken string, templateVars map[string]string) error {
	err := do(ctx, kcConfig, config, callback, username, usertoken, templateVars, true)
	if err != nil {
		return err
	}
	return nil
}

func do(ctx context.Context, kcConfig keycloak.Config, config Config, callback Callback, username, usertoken string, templateVars map[string]string, update bool) error {
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

	extension := "openshift.yml"
	if KubernetesMode() {
		extension = "kubernetes.yml"

		keycloakUrl, err := FindKeyCloakURL(config)
		if err != nil {
			return fmt.Errorf("Could not find the KeyCloak URL: %v", err)
		}
		vars[varKeycloakURL] = keycloakUrl
	}

	userProjectT, err := loadTemplate(config, "fabric8-online-user-project-"+extension)
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

	projectT, err := loadTemplate(config, "fabric8-online-team-"+extension)
	if err != nil {
		return err
	}

	jenkinsT, err := loadTemplate(config, "fabric8-online-jenkins-"+extension)
	if err != nil {
		return err
	}

	cheT, err := loadTemplate(config, "fabric8-online-che-"+extension)
	if err != nil {
		return err
	}

	err = executeNamespaceSync(string(userProjectT), vars, userOpts)
	if err != nil {
		return err
	}

	var channels []chan error
	syncErrorChannel := make(chan error)
	channels = append(channels, syncErrorChannel)

	// TODO have kubernetes versions of these!
	if !KubernetesMode() {
		err = executeNamespaceSync(string(userProjectCollabT), vars, masterOpts.WithNamespace(name))
		if err != nil {
			syncErrorChannel <- err
		}
		err = executeNamespaceSync(string(userProjectRolesT), vars, userOpts.WithNamespace(name))
		if err != nil {
			syncErrorChannel <- err
		}
	}

	{
		lvars := clone(vars)
		lvars[varProjectDisplayName] = lvars[varProjectName]

		err = executeNamespaceSync(string(projectT), lvars, masterOpts.WithNamespace(name))
		if err != nil {
			syncErrorChannel <- err
		}
	}
	// Quotas needs to be applied before we attempt to install the resources on OSO
	osoQuotas := true
	disableOsoQuotasFlag := os.Getenv("DISABLE_OSO_QUOTAS")
	if disableOsoQuotasFlag == "true" {
		osoQuotas = false
	}
	if osoQuotas && !KubernetesMode() {
		jenkinsQuotasT, err := loadTemplate(config, "fabric8-online-jenkins-quotas-oso-"+extension)
		if err != nil {
			return err
		}
		cheQuotasT, err := loadTemplate(config, "fabric8-online-che-quotas-oso-"+extension)
		if err != nil {
			return err
		}

		{
			lvars := clone(vars)
			nsname := fmt.Sprintf("%v-jenkins", name)
			lvars[varProjectNamespace] = vars[varProjectName]
			err := executeNamespaceSync(string(jenkinsQuotasT), lvars, masterOpts.WithNamespace(nsname))
			if err != nil {
				syncErrorChannel <- err
			}
		}
		{
			lvars := clone(vars)
			nsname := fmt.Sprintf("%v-che", name)
			lvars[varProjectNamespace] = vars[varProjectName]
			err := executeNamespaceSync(string(cheQuotasT), lvars, masterOpts.WithNamespace(nsname))
			if err != nil {
				syncErrorChannel <- err
			}
		}
	}

	{
		lvars := clone(vars)
		nsname := fmt.Sprintf("%v-jenkins", name)
		lvars[varProjectNamespace] = vars[varProjectName]
		if update {
			output, err := executeNamespaceCMD(string(jenkinsT), lvars, masterOpts.WithNamespace(nsname))
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"output":    output,
					"namespace": nsname,
					"error":     err,
				}, "failed")

				syncErrorChannel <- err
			}
			log.Info(ctx, map[string]interface{}{
				"output":    output,
				"namespace": nsname,
			}, "applied")
		} else {
			channels = append(channels, executeNamespaceAsync(string(jenkinsT), lvars, masterOpts.WithNamespace(nsname)))
		}

	}
	if KubernetesMode() {
		exposeT, err := loadTemplate(config, "fabric8-online-expose-kubernetes.yml")
		if err != nil {
			return err
		}
		exposeVars, err := LoadExposeControllerVariables(config)
		if err != nil {
			return err
		}

		{
			lvars := clone(vars)
			for k, v := range exposeVars {
				lvars[k] = v
			}
			nsname := fmt.Sprintf("%v-jenkins", name)
			lvars[varProjectNamespace] = vars[varProjectName]
			err := executeNamespaceSync(string(exposeT), lvars, masterOpts.WithNamespace(nsname))
			if err != nil {
				syncErrorChannel <- err
			}
		}
		{
			lvars := clone(vars)
			for k, v := range exposeVars {
				lvars[k] = v
			}
			nsname := fmt.Sprintf("%v-che", name)
			lvars[varProjectNamespace] = vars[varProjectName]
			err := executeNamespaceSync(string(exposeT), lvars, masterOpts.WithNamespace(nsname))
			if err != nil {
				syncErrorChannel <- err
			}
		}
	}
	{
		lvars := clone(vars)
		nsname := fmt.Sprintf("%v-che", name)
		lvars[varProjectNamespace] = vars[varProjectName]
		if update {
			output, err := executeNamespaceCMD(string(cheT), lvars, masterOpts.WithNamespace(nsname))
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"output":    output,
					"namespace": nsname,
					"error":     err,
				}, "failed")
				syncErrorChannel <- err
			}
			log.Info(ctx, map[string]interface{}{
				"output":    output,
				"namespace": nsname,
			}, "applied")
		} else {
			channels = append(channels, executeNamespaceAsync(string(cheT), lvars, masterOpts.WithNamespace(nsname)))
		}

	}

	if KubernetesMode() {
		// lets try create the KeyCloak client for the jenkins service
		jenkinsNS := fmt.Sprintf("%v-jenkins", name)
		_, err = EnsureKeyCloakHasJenkinsRedirectURL(config, kcConfig, jenkinsNS)
		if err != nil {
			syncErrorChannel <- fmt.Errorf("Failed to register redirectUri into KeyCloak for jenkins in %s due to %v", jenkinsNS, err)
		}
		/*
			} else {
				channels = append(channels, EnsureKeyCloakHasJenkinsRedirectURLAsync(config, kcConfig, jenkinsNS))
			}
		*/
	}
	close(syncErrorChannel)
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
		case "fabric8-online-jenkins-quotas-oso-openshift.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-jenkins-quotas-oso/$TEAM_VERSION/fabric8-online-jenkins-quotas-oso-$TEAM_VERSION-openshift.yml"
		case "fabric8-online-che-openshift.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-che/$TEAM_VERSION/fabric8-online-che-$TEAM_VERSION-openshift.yml"
		case "fabric8-online-che-quotas-oso-openshift.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-che-quotas-oso/$TEAM_VERSION/fabric8-online-che-quotas-oso-$TEAM_VERSION-openshift.yml"
		case "fabric8-online-team-kubernetes.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-team/$TEAM_VERSION/fabric8-online-team-$TEAM_VERSION-k8s-template.yml"
		case "fabric8-online-jenkins-kubernetes.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-jenkins/$TEAM_VERSION/fabric8-online-jenkins-$TEAM_VERSION-k8s-template.yml"
		case "fabric8-online-che-kubernetes.yml":
			url = "http://central.maven.org/maven2/io/fabric8/online/packages/fabric8-online-che/$TEAM_VERSION/fabric8-online-che-$TEAM_VERSION-k8s-template.yml"
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

func executeNamespaceCMD(template string, vars map[string]string, opts ApplyOptions) (string, error) {
	t, err := Process(template, vars)
	if err != nil {
		return "", err
	}

	cmdName := "/usr/bin/sh"
	hostVerify := ""
	flag := os.Getenv("KEYCLOAK_SKIP_HOST_VERIFY")
	if strings.ToLower(flag) == "true" {
		hostVerify = " --insecure-skip-tls-verify=true"
	}
	serverFlag := "--server=" + opts.MasterURL + hostVerify
	if KubernetesMode() {
		serverFlag = "--local=true"
	}

	cmdArgs := []string{"-c", "oc process -f - " + serverFlag + " --token=" + opts.Token + " --namespace=" + opts.Namespace + " | oc apply -f -  --overwrite=true --force=true --server=" + opts.MasterURL + hostVerify + " --token=" + opts.Token + " --namespace=" + opts.Namespace}

	var buf bytes.Buffer
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, t)

	}()
	if err := cmd.Start(); err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		return buf.String(), err
	}

	return buf.String(), nil
}

func KubernetesMode() bool {
	k8sMode := os.Getenv("F8_KUBERNETES_MODE")
	return k8sMode == "true"
}

func clone(maps map[string]string) map[string]string {
	maps2 := make(map[string]string)
	for k2, v2 := range maps {
		maps2[k2] = v2
	}
	return maps2
}
