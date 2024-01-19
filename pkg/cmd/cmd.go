/*
Copyright 2022-2024 EscherCloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/pkg/cmd/create"
	"github.com/spjmurray/unikorn/pkg/cmd/delete"
	"github.com/spjmurray/unikorn/pkg/cmd/get"
	"github.com/spjmurray/unikorn/pkg/constants"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	//nolint:gochecknoglobals
	rootLongDesc = templates.LongDesc(`
	EscherCloudAI Kubernetes Provisioning.

	This CLI toolset provides dynamic provisioning of Kubernetes clusters
	and Cluster API control planes.  It also provides various Kubernetes
	cluster life-cycle management functions.  For additional details on
	how the individual components operatate, see the individual 'create'
	help topics.`)
)

// newRootCommand returns the root command and all its subordinates.
// This sets global flags for standard Kubernetes configuration options
// such as --kubeconfig, --context, --namespace, etc.
func newRootCommand() *cobra.Command {
	configFlags := genericclioptions.NewConfigFlags(true)

	f := cmdutil.NewFactory(configFlags)

	cmd := &cobra.Command{
		Use:   constants.Application,
		Short: "EscherCloudAI Kubernetes Provisioning.",
		Long:  rootLongDesc,
	}

	configFlags.AddFlags(cmd.PersistentFlags())

	commands := []*cobra.Command{
		newVersionCommand(),
		create.NewCreateCommand(f),
		delete.NewDeleteCommand(f),
		get.NewGetCommand(f),
	}

	cmd.AddCommand(commands...)

	return cmd
}

// Generate creates a hierarchy of cobra commands for the application.  It can
// also be used to walk the structure and generate HTML documentation for example.
func Generate() *cobra.Command {
	return newRootCommand()
}
