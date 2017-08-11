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
package util

import (
	"os/exec"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	// Minikube context name
	Minikube = "minikube"
	// Minishift context name
	Minishift = "minishift"
	// CDK seems like an odd context name, lets try it for now in the absence of anything else
	CDK = "default/10-1-2-2:8443/admin"
)

// IsMini returns true if we are using a minikube or minishift context
func IsMini() (bool, error) {
	currentContext, err := GetCurrentContext()

	if err != nil {
		return false, err
	}

	if currentContext == Minikube || IsMiniShift(currentContext) {
		return true, nil
	}
	return false, nil
}

func IsMiniShift(currentContext string) bool {
	out, err := exec.Command("minishift", "ip").Output()
	if err != nil || out == nil {
		return false
	}
	ip := strings.Replace(string(out), ".", "-", -1)
	ip = strings.TrimSpace(ip)

	return currentContext == Minishift || strings.Contains(currentContext, ip)
}

// GetMiniType returns whether this is a minishift or minikube including which one
func GetMiniType() (string, bool, error) {
	currentContext, err := GetCurrentContext()

	if err != nil {
		return "", false, err
	}

	if currentContext == Minikube {
		return Minikube, true, nil
	}
	if IsMiniShift(currentContext) {
		return Minishift, true, nil
	}
	return currentContext, false, nil
}

// GetCurrentContext gets the current context from local config
func GetCurrentContext() (string, error) {
	clientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	return clientConfig.CurrentContext, err
}
