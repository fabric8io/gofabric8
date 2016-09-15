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
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return false, err
	}
	context := strings.TrimSpace(string(out))
	if context == Minikube || context == Minishift {
		return true, nil
	}
	return false, nil
}

// GetConfigContext returns the context name
func GetConfigContext() (string, error) {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return "", err
	}
	context := strings.TrimSpace(string(out))
	return context, nil
}

// GetMiniCLI returns whether this is a minishift or minikube including which one
func GetMiniType() (string, bool, error) {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return "", false, err
	}

	context := strings.TrimSpace(string(out))
	if context == Minikube || context == Minishift {
		return context, true, nil
	}
	return context, false, nil
}
