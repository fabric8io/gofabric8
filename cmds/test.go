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

type testPipelineFlags struct {
	confirm bool
}

// NewCmdTest performs a test pipeline to check an installation
func NewCmdTest(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "test",
		Short:   "Runs the end to end system tests",
		Long:    `Runs the end to end system tests`,
		Aliases: []string{"content-repository"},

		Run: func(cmd *cobra.Command, args []string) {
			p := testPipelineFlags{}
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			err := p.testPipeline(f)
			if err != nil {
				util.Fatalf("%s\n", err)
			}
			return
		},
	}
	return cmd
}

func (p *testPipelineFlags) testPipeline(f cmdutil.Factory) error {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	oc, _ := client.NewOpenShiftClient(cfg)
	initSchema()

	userNS, err := detectCurrentUserNamespace(ns, c, oc)
	if err != nil {
		return err
	}
	jenkinsNS := fmt.Sprintf("%s-jenkins", userNS)

	util.Info("Starting the end to end test pipeline for tenant: ")
	util.Successf("%s\n", userNS)

	err = ensureDeploymentOrDCHasReplicas(c, oc, jenkinsNS, "jenkins", 1)
	if err != nil {
		return err
	}

	testPipelineJob := "internal-end-to-end-tests"

	// lets check if we have a ConfigMap for the jenkins job
	buildConfigSpec := buildapi.BuildConfigSpec{
		RunPolicy: buildapi.BuildRunPolicySerial,
		CommonSpec: buildapi.CommonSpec{
			Strategy: buildapi.BuildStrategy{
				//Type: buildapi.JenkinsPipeline,
				JenkinsPipelineStrategy: &buildapi.JenkinsPipelineBuildStrategy{
					JenkinsfilePath: "Jenkinsfile",
				},
			},
			Source: buildapi.BuildSource{
				Git: &buildapi.GitBuildSource{
					Ref: "master",
					//URI: "https://github.com/jstrachan/fabric8-ui-e2e-test.git",
					URI: "https://github.com/fabric8io/fabric8-test.git",
				},
			},
		},
	}
	create := false
	operation := "update"
	bc, err := oc.BuildConfigs(userNS).Get(testPipelineJob)
	if err != nil {
		bc = &buildapi.BuildConfig{
			ObjectMeta: api.ObjectMeta{
				Namespace:   userNS,
				Name:        testPipelineJob,
				Annotations: map[string]string{"jenkins.openshift.org/generated-by": "gofabric8"},
			},
			Spec: buildConfigSpec,
		}
		create = true
		operation = "create"
	}
	if create {
		_, err = oc.BuildConfigs(userNS).Create(bc)
	} else {
		bc.Spec = buildConfigSpec
		_, err = oc.BuildConfigs(userNS).Update(bc)
	}
	if err != nil {
		return fmt.Errorf("Failed to %s BuildConfig %s in namespace %s due to: %s", operation, testPipelineJob, userNS, err)
	}
	request := buildapi.BuildRequest{
		ObjectMeta: api.ObjectMeta{
			Name: testPipelineJob,
		},
	}
	build, err := oc.BuildConfigs(userNS).Instantiate(&request)
	if err != nil {
		return fmt.Errorf("Failed to instantiate BuildConfig %s in namespace %s due to: %s", testPipelineJob, userNS, err)
	}
	util.Info("Started build to run the end to end tests: ")
	util.Successf("%s\n", build.Name)
	return watchAndWaitForBuild(oc, userNS, build.Name, time.Hour*2)
}
