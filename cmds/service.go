/**
 * Copyright (C) 2015 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package cmds

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	kubeApi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	urlCommandFlag       = "url"
	retryFlag            = "retry"
	namespaceCommandFlag = "namespace"
	namespaceFileFlag    = "namespace-file"
	exposeURLAnnotation  = "fabric8.io/exposeUrl"
)

// NewCmdService looks up the external service address and opens the URL
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go
func NewCmdService(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Opens the specified Kubernetes service in your browser",
		Long:  `Opens the specified Kubernetes service in your browser`,

		Run: func(cmd *cobra.Command, args []string) {
			var err error
			c, cfg := client.NewClient(f)

			ns := cmd.Flags().Lookup(namespaceCommandFlag).Value.String()
			if ns == "" {
				ns, err = defaultNamespace(cmd, f)
				cmdutil.CheckErr(err)
			}
			typeOfMaster := util.TypeOfMaster(c)

			printURL := cmd.Flags().Lookup(urlCommandFlag).Value.String() == "true"
			retry := cmd.Flags().Lookup(retryFlag).Value.String() == "true"
			if len(args) != 1 {
				util.Fatalf("Please choose a service, found %v arguments\n", len(args))
			}

			if typeOfMaster == util.OpenShift {
				openServiceWithRoute(ns, args[0], cfg, printURL, retry)
			} else {
				openService(ns, args[0], c, printURL, retry)
			}
		},
	}
	cmd.PersistentFlags().StringP(namespaceCommandFlag, "n", "default", "The service namespace")
	cmd.PersistentFlags().BoolP(urlCommandFlag, "u", false, "Display the kubernetes service exposed URL in the CLI instead of opening it in the default browser")
	cmd.PersistentFlags().Bool(retryFlag, true, "Retries to find the service if its not available just yet")
	return cmd
}

func searchForRouteNameURL(name, ns string, oc *oclient.Client) (url string, err error) {
	// TODO: why this doesn't work on api.starter-us-east-2.openshift.com ?
	routes, err := oc.Routes(ns).List(kubeApi.ListOptions{})
	if err != nil {
		return
	}

	for _, item := range routes.Items {
		url = "http"
		if item.Spec.TLS != nil {
			url = "https"
		}
		url += fmt.Sprintf("://%s", item.Spec.Host)
		if item.Spec.Port != nil {
			url += fmt.Sprintf(":%d", item.Spec.Port.TargetPort.IntValue())
		}
		if item.Name == name {
			//TODO(chmou): Check External Service if ready and then print!
			return
		}
	}
	return "", nil
}

func openServiceWithRoute(ns string, serviceName string, cfg *restclient.Config, printURL bool, retry bool) {
	items := []string{ns}
	found := false
	oc, _ := client.NewOpenShiftClient(cfg)
	initSchema()

	projects, err := oc.Projects().List(kubeApi.ListOptions{})
	cmdutil.CheckErr(err)
	for _, todoNS := range projects.Items {
		items = append(items, todoNS.Name)
	}

	for _, todoNS := range items {
		url, err := searchForRouteNameURL(serviceName, todoNS, oc)
		if err != nil {
			// NB: Do not fail here as we want the auto detection to go on if we
			// had the default namespaces
			util.Info(err.Error())
		}
		if url != "" {
			if printURL {
				util.Successf("%s\n", url)
			} else {
				util.Successf("\nOpening URL %s\n", url)
				browser.OpenURL(url)
			}
			found = true
			break
		}
	}

	if !found {
		util.Errorf("No service %s in namespace %s\n", serviceName, ns)
	}
}

func openService(ns string, serviceName string, c *clientset.Clientset, printURL bool, retry bool) {
	if retry {
		if err := RetryAfter(1200, func() error { return CheckExternalService(ns, serviceName, c) }, 10*time.Second); err != nil {
			util.Errorf("Could not find finalized endpoint being pointed to by %s: %v", serviceName, err)
			os.Exit(1)
		}
	}
	svcs, err := c.Services(ns).List(kubeApi.ListOptions{})
	if err != nil {
		util.Errorf("No services found %v\n", err)
	}
	found := false
	for _, service := range svcs.Items {
		if serviceName == service.Name {

			url := service.ObjectMeta.Annotations[exposeURLAnnotation]

			if printURL {
				util.Successf("%s\n", url)
			} else {
				util.Successf("\nOpening URL %s\n", url)
				browser.OpenURL(url)
			}
			found = true
			break
		}
	}
	if !found {
		util.Errorf("No service %s in namespace %s\n", serviceName, ns)
	}
}

// FindServiceURL returns the external service URL waiting a little bit for it to show up
func FindServiceURL(ns string, serviceName string, c *clientset.Clientset, retry bool) string {
	answer := ""
	if retry {
		if err := RetryAfter(1200, func() error { return CheckServiceExists(ns, serviceName, c) }, 10*time.Second); err != nil {
			util.Errorf("Could not find finalized endpoint being pointed to by %s: %v", serviceName, err)
			os.Exit(1)
		}
	}
	svc, err := c.Services(ns).Get(serviceName)
	if err != nil {
		util.Errorf("No service %s found %v\n", serviceName, err)
	}
	url := svc.ObjectMeta.Annotations[exposeURLAnnotation]
	if len(url) > 0 {
		answer = url
	}
	return answer
}

// GetServiceURL returns the external service URL or returns the empty string if it cannot be found
func GetServiceURL(ns string, serviceName string, c *clientset.Clientset) string {
	answer := ""
	svc, err := c.Services(ns).Get(serviceName)
	if err != nil {
		return answer
	}
	ann := svc.ObjectMeta.Annotations
	if ann != nil {
		return ann[exposeURLAnnotation]
	}
	return answer
}

// GetServiceOrRouteURL returns the external service URL or the route URL returns the empty string if it cannot be found
func GetServiceOrRouteURL(ns string, serviceName string, c *clientset.Clientset, oc *oclient.Client, protocol string) string {
	answer := GetServiceURL(ns, serviceName, c)
	if len(answer) == 0 {
		r, err := oc.Routes(ns).Get(serviceName)
		if err == nil {
			return protocol + r.Spec.Host
		}
	}
	return answer
}

// WaitForService waits for a service and its endpoint to be ready
func WaitForService(ns string, serviceName string, c *clientset.Clientset) {
	if err := RetryAfter(1200, func() error { return CheckService(ns, serviceName, c) }, 10*time.Second); err != nil {
		util.Errorf("Could not find finalized endpoint being pointed to by %s: %v", serviceName, err)
		os.Exit(1)
	}
}

// CheckServiceExists waits for the specified service to have an external URL
func CheckServiceExists(ns string, service string, c *clientset.Clientset) error {
	svc, err := c.Services(ns).Get(service)
	if err != nil {
		return err
	}
	url := svc.ObjectMeta.Annotations[exposeURLAnnotation]
	if url == "" {
		util.Info(".")
		return errors.New("")
	}
	return nil
}

// CheckExternalService waits for the specified service to be ready by returning an error until the service is up
// The check is done by polling the endpoint associated with the service and when the endpoint exists, returning no error->service-online
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go#L89
func CheckExternalService(ns string, service string, c *clientset.Clientset) error {
	svc, err := c.Services(ns).Get(service)
	if err != nil {
		return err
	}
	url := svc.ObjectMeta.Annotations[exposeURLAnnotation]
	if url == "" {
		return errors.New("No external URL annotation")
	}
	endpoints := c.Endpoints(ns)
	if endpoints == nil {
		util.Errorf("No endpoints found in namespace %s\n", ns)
	}
	endpoint, err := endpoints.Get(service)
	if err != nil {
		util.Errorf("No endpoints found for service %s\n", service)
		return err
	}
	return CheckEndpointReady(endpoint)
}

// CheckService waits for the specified service to be ready by returning an error until the service is up
// The check is done by polling the endpoint associated with the service and when the endpoint exists, returning no error->service-online
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go#L89
func CheckService(ns string, service string, c *clientset.Clientset) error {
	endpoints := c.Endpoints(ns)
	if endpoints == nil {
		util.Errorf("No endpoints found in namespace %s\n", ns)
	}
	endpoint, err := endpoints.Get(service)
	if err != nil {
		util.Errorf("No endpoints found for service %s\n", service)
		return err
	}
	return CheckEndpointReady(endpoint)
}

//CheckEndpointReady checks that the kubernetes endpoint is ready
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go#L101
func CheckEndpointReady(endpoint *kubeApi.Endpoints) error {
	if len(endpoint.Subsets) == 0 {
		fmt.Fprintf(os.Stderr, ".")
		return fmt.Errorf("Endpoint for service is not ready yet\n")
	}
	for _, subset := range endpoint.Subsets {
		if len(subset.NotReadyAddresses) != 0 {
			fmt.Fprintf(os.Stderr, "Waiting, endpoint for service is not ready yet...\n")
			return fmt.Errorf("Endpoint for service is not ready yet\n")
		}
	}
	return nil
}

//WaitForExternalIPAddress will wait for loadbalancers to update the service and return it's external ip address
func WaitForExternalIPAddress(ns string, serviceName string, c *clientset.Clientset) (address string, err error) {

	if err := RetryAfter(1200, func() error { return HasExternalIP(ns, serviceName, c) }, 10*time.Second); err != nil {
		util.Errorf("Could not find external IP for %s: %v", serviceName, err)
		os.Exit(1)
	}
	svc, err := c.Services(ns).Get(serviceName)
	if err != nil {
		return "", err
	}
	if svc.Status.LoadBalancer.Ingress[0].IP != "" {
		return svc.Status.LoadBalancer.Ingress[0].IP, nil
	}
	return svc.Status.LoadBalancer.Ingress[0].Hostname, nil
}

//HasExternalIP checks if a service has an external ip address
func HasExternalIP(ns string, serviceName string, c *clientset.Clientset) error {
	svc, err := c.Services(ns).Get(serviceName)
	if err != nil {
		return err
	}
	if len(svc.Status.LoadBalancer.Ingress) > 0 && (svc.Status.LoadBalancer.Ingress[0].IP != "" || svc.Status.LoadBalancer.Ingress[0].Hostname != "") {
		return nil
	}
	return fmt.Errorf("Service has no external ip or hostname yet\n")
}

func Retry(attempts int, callback func() error) (err error) {
	return RetryAfter(attempts, callback, 0)
}

func RetryAfter(attempts int, callback func() error, d time.Duration) (err error) {
	m := MultiError{}
	for i := 0; i < attempts; i++ {
		err = callback()
		if err == nil {
			return nil
		}
		m.Collect(err)
		time.Sleep(d)
	}
	return m.ToError()
}

type MultiError struct {
	Errors []error
}

func (m *MultiError) Collect(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

func (m MultiError) ToError() error {
	if len(m.Errors) == 0 {
		return nil
	}

	errStrings := []string{}
	for _, err := range m.Errors {
		errStrings = append(errStrings, err.Error())
	}
	return fmt.Errorf(strings.Join(errStrings, "\n"))
}
