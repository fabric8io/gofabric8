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
	"os"

	"github.com/spf13/cobra"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/pflag"
)

type Result string

const (
	Success Result = "✔"
	Failure Result = "✘"

	// cmd flags
	yesFlag = "yes"
    hostPathFlag = "host-path"
	nameFlag = "name"
	domainFlag = "domain"


	DefaultDomain = "vagrant.f8"

)

func defaultDomain() string {
	defaultDomain := os.Getenv("KUBERNETES_DOMAIN")
	if defaultDomain == "" {
		defaultDomain = DefaultDomain
	}
	return defaultDomain
}
func missingFlag(cmd *cobra.Command, name string)(Result, error) {
	util.Errorf("No option -%s specified!\n", hostPathFlag)
	text := cmd.Name()
	parent := cmd.Parent()
	if parent != nil {
		text = parent.Name() + " " + text
	}
	util.Infof("Please try something like: %s --%s='some value' ...\n\n", text, hostPathFlag)
	return Failure, nil
}

func confirmAction(flags *pflag.FlagSet) bool {
	if flags.Lookup(yesFlag).Value.String() == "false" {
		util.Info("Continue? [Y/n] ")
		cont := util.AskForConfirmation(true)
		if !cont {
			util.Fatal("Cancelled...\n")
			return false
		}
	}
	return true
}
