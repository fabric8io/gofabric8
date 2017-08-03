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
	"time"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	buildapi "github.com/openshift/origin/pkg/build/api"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type cleanUpMavenLocalRepoFlags struct {
	confirm bool
}

// NewCmdCleanUpMavenLocalRepo delete files in the tenants content repository
func NewCmdCleanUpMavenLocalRepo(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "maven-local-repo",
		Short:   "Hard delete the local maven repository files",
		Long:    `Hard delete the local maven repository files. These are a cache used to download maven repository content to speed up your builds. But can be periodically deleted to reduce disk space.`,
		Aliases: []string{"content-repository"},

		Run: func(cmd *cobra.Command, args []string) {
			p := cleanUpMavenLocalRepoFlags{}
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			err := p.cleanMavenLocalRepo(f)
			if err != nil {
				util.Fatalf("%s\n", err)
			}
			return
		},
	}
	return cmd
}

func (p *cleanUpMavenLocalRepoFlags) cleanMavenLocalRepo(f *cmdutil.Factory) error {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	oc, _ := client.NewOpenShiftClient(cfg)
	initSchema()

	userNS, err := detectCurrentUserNamespace(ns, c, oc)
	if err != nil {
		return err
	}
	jenkinsNS := fmt.Sprintf("%s-jenkins", userNS)

	if !p.confirm {
		confirm := ""
		util.Warn("WARNING this is destructive and will remove the local maven repository cache which will reduce disk space but slow down your next build\n")
		util.Info("for your tenant: ")
		util.Successf("%s", userNS)
		util.Info(" running in namespace: ")
		util.Successf("%s\n", jenkinsNS)
		util.Warn("\nContinue [y/N]: ")
		fmt.Scanln(&confirm)
		if confirm != "y" {
			util.Warn("Aborted\n")
			return nil
		}
	}
	util.Info("Cleaning local maven repository for tenant: ")
	util.Successf("%s", userNS)
	util.Info(" running in namespace: ")
	util.Successf("%s\n", jenkinsNS)

	err = ensureDeploymentOrDCHasReplicas(c, oc, jenkinsNS, "jenkins", 1)
	if err != nil {
		return err
	}

	cleanMavenLocalRepoJob := "clean-mvn-local-repo-" + fmt.Sprintf("%x", time.Now().Unix())

	// lets check if we have a ConfigMap for the jenkins job
	buildConfigSpec := buildapi.BuildConfigSpec{
		RunPolicy: buildapi.BuildRunPolicySerial,
		CommonSpec: buildapi.CommonSpec{
			ServiceAccount: "jenkins",
			Strategy: buildapi.BuildStrategy{
				//Type: buildapi.JenkinsPipeline,
				JenkinsPipelineStrategy: &buildapi.JenkinsPipelineBuildStrategy{
					Jenkinsfile: `@Library('github.com/fabric8io/fabric8-pipeline-library@master')
              def dummy
              mavenNode{
                container('maven'){
                  echo "clearing local maven repository at /root/.mvnrepository"
                  sh 'rm -rf /root/.mvnrepository/*'
                  sh 'du -hc /root/.mvnrepository'
                }
              }`,
				},
			},
		},
	}
	create := false
	bc, err := oc.BuildConfigs(userNS).Get(cleanMavenLocalRepoJob)
	if err != nil {
		create = true
	}
	if create {
		newBC := buildapi.BuildConfig{
			ObjectMeta: api.ObjectMeta{
				Namespace: userNS,
				Name:      cleanMavenLocalRepoJob,
			},
			Spec: buildConfigSpec,
		}
		_, err = oc.BuildConfigs(userNS).Create(&newBC)
		if err != nil {
			return fmt.Errorf("Failed to create BuildConfig %s in namespace %s due to: %s", cleanMavenLocalRepoJob, userNS, err)
		}
	} else {
		bc.Spec = buildConfigSpec
		_, err = oc.BuildConfigs(userNS).Update(bc)
		if err != nil {
			util.Infof("Failed to update BuildConfig %s in namespace %s due to: %s\n", cleanMavenLocalRepoJob, userNS, err)
		}
	}
	time.Sleep(time.Second * 2)

	request := buildapi.BuildRequest{
		ObjectMeta: api.ObjectMeta{
			Name: cleanMavenLocalRepoJob,
		},
	}
	build, err := oc.BuildConfigs(userNS).Instantiate(&request)
	if err != nil {
		return fmt.Errorf("Failed to instantiate BuildConfig %s in namespace %s due to: %s", cleanMavenLocalRepoJob, userNS, err)
	}
	util.Info("Started build to clear down the local maven repository in the OpenShift Build: ")
	util.Successf("%s\n", build.Name)
	err = watchAndWaitForBuild(oc, userNS, build.Name, time.Minute*15)

	runCommand("oc", "delete", "bc", cleanMavenLocalRepoJob, "-n", userNS, "--cascade=true", "--grace-period=-10")
	if err != nil {
		return fmt.Errorf("Failed to complete build in 15 minutes due to %s", err)
	}
	return nil
}
