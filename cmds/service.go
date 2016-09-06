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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	kubeApi "k8s.io/kubernetes/pkg/api"
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	urlCommandFlag       = "url"
	namespaceCommandFlag = "namespace"
	exposeURLAnnotation  = "fabric8.io/exposeUrl"
)

// NewCmdService looks up the external service address and opens the URL
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go
func NewCmdService(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Opens the specified Kubernetes service in your browser",
		Long:  `Opens the specified Kubernetes service in your browser`,

		Run: func(cmd *cobra.Command, args []string) {
			c, _ := client.NewClient(f)

			ns := cmd.Flags().Lookup(namespaceCommandFlag).Value.String()
			if ns == "" {
				ns, _, _ = f.DefaultNamespace()
			}

			serviceName := args[0]
			if err := RetryAfter(20, func() error { return CheckService(ns, serviceName, c) }, 6*time.Second); err != nil {
				util.Errorf("Could not find finalized endpoint being pointed to by %s: %v", serviceName, err)
				os.Exit(1)
			}

			svcs, err := c.Services(ns).List(api.ListOptions{})
			if err != nil {
				util.Errorf("No services found %v\n", err)
			}
			found := false
			for _, service := range svcs.Items {
				if serviceName == service.Name {

					url := service.ObjectMeta.Annotations[exposeURLAnnotation]
					printURL := cmd.Flags().Lookup(urlCommandFlag).Value.String() == "true"
					if printURL {
						util.Successf("%s\n", url)
					} else {
						util.Successf("Opening URL %s\n", url)
						browser.OpenURL(url)
					}
					found = true
					break
				}
			}
			if !found {
				util.Errorf("No service %s in namespace %s\n", serviceName, ns)
			}
		},
	}
	cmd.PersistentFlags().StringP(namespaceCommandFlag, "n", "default", "The service namespace")
	cmd.PersistentFlags().BoolP(urlCommandFlag, "u", false, "Display the kubernetes service exposed URL in the CLI instead of opening it in the default browser")
	return cmd
}

// CheckService waits for the specified service to be ready by returning an error until the service is up
// The check is done by polling the endpoint associated with the service and when the endpoint exists, returning no error->service-online
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go#L89
func CheckService(ns string, service string, c *k8sclient.Client) error {
	endpoints := c.Endpoints(ns)
	if endpoints == nil {
		util.Errorf("No endpoints found in namespace %s\n", ns)
	}
	endpoint, err := endpoints.Get(service)
	if err != nil {
		return err
	}
	return CheckEndpointReady(endpoint)
}

//CheckEndpointReady checks that the kubernetes endpoint is ready
// Credits: https://github.com/kubernetes/minikube/blob/v0.9.0/cmd/minikube/cmd/service.go#L101
func CheckEndpointReady(endpoint *kubeApi.Endpoints) error {
	if len(endpoint.Subsets) == 0 {
		fmt.Fprintf(os.Stderr, "Waiting, endpoint for service is not ready yet...\n")
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
