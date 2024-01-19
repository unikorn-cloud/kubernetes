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

package get

import (
	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/pkg/cmd/get/kubeconfig"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// NewGetCommand returns a command that can list all resources, or get information
// about a single one.
func NewGetCommand(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get and list Unikorn resources",
		Long:  "Get and list Unikorn resources",
	}

	commands := []*cobra.Command{
		newGetProjectCommand(f),
		newGetControlPlaneCommand(f),
		newGetClusterCommand(f),
		kubeconfig.NewGetKubeconfigCommand(f),
	}

	cmd.AddCommand(commands...)

	return cmd
}
