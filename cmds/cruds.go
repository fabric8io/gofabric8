/**
 * Copyright (C) 2017 Red Hat, Inc.
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
	"io"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	longhelp = `You must specify the type of resource to get. Valid resource types include:
* cluster.
* environ.
`
)

func NewCmdCreate(f *cmdutil.Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a resource type",
		Long:  longhelp,
	}

	cmd.AddCommand(NewCmdCreateEnviron(f))
	return cmd
}

func NewCmdDelete(f *cmdutil.Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a resource type",
		Long:  longhelp,
	}

	cmd.AddCommand(NewCmdDeleteCluster(f))
	cmd.AddCommand(NewCmdDeleteEnviron(f))
	return cmd
}

func NewCmdCleanUp(f *cmdutil.Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up a resource type without deleting it",
		Long:  longhelp,
	}

	cmd.AddCommand(NewCmdCleanUpApp(f))
	cmd.AddCommand(NewCmdCleanUpContentRepository(f))
	cmd.AddCommand(NewCmdCleanUpJenkins(f))
	cmd.AddCommand(NewCmdCleanUpMavenLocalRepo(f))
	cmd.AddCommand(NewCmdCleanUpSystem(f))

	// TODO
	// cmd.AddCommand(NewCmdCleanUpTenant(f))
	return cmd
}

func NewCmdGet(f *cmdutil.Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a resource type",
		Long:  `get a resource type`,
	}

	cmd.AddCommand(NewCmdGetEnviron(f, out))
	return cmd
}
