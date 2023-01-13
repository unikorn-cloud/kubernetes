/*
Copyright 2022 EscherCloud.

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
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/eschercloudai/unikorn/generated/clientset/unikorn"
	"github.com/eschercloudai/unikorn/pkg/cmd/util"
	"github.com/eschercloudai/unikorn/pkg/cmd/util/flags"
	"github.com/eschercloudai/unikorn/pkg/provisioners/vcluster"

	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type getKubeConfigOptions struct {
	// controlPlaneFlags define control plane scoping.
	controlPlaneFlags flags.ControlPlaneFlags

	// client is the Kubernetes v1 client.
	client kubernetes.Interface

	// unikornClient gives access to our custom resources.
	unikornClient unikorn.Interface
}

// addFlags registers create cluster options flags with the specified cobra command.
func (o *getKubeConfigOptions) addFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.controlPlaneFlags.AddFlags(f, cmd)
}

// complete fills in any options not does automatically by flag parsing.
func (o *getKubeConfigOptions) complete(f cmdutil.Factory, _ []string) error {
	var err error

	if o.client, err = f.KubernetesClientSet(); err != nil {
		return err
	}

	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}

	if o.unikornClient, err = unikorn.NewForConfig(config); err != nil {
		return err
	}

	return nil
}

// validate validates any tainted input not handled by complete() or flags
// processing.
func (o *getKubeConfigOptions) validate() error {
	return nil
}

// run executes the command.
func (o *getKubeConfigOptions) run() error {
	namespace, err := o.controlPlaneFlags.GetControlPlaneNamespace(context.TODO(), o.unikornClient)
	if err != nil {
		return err
	}

	configPath, cleanup, err := vcluster.WriteConfig(context.TODO(), vcluster.NewKubectlGetter(o.client), namespace)
	if err != nil {
		return err
	}

	defer cleanup()

	f, err := os.Open(configPath)
	if err != nil {
		return err
	}

	out, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	f.Close()

	fmt.Println(string(out))

	return nil
}

// newGetKubeConfigCommand creates a command that gets a Cluster API control plane.
func newGetKubeConfigCommand(f cmdutil.Factory) *cobra.Command {
	o := getKubeConfigOptions{}

	cmd := &cobra.Command{
		Use:               "kubeconfig",
		Short:             "Delete a Kubernetes cluster",
		Long:              "Delete a Kubernetes cluster",
		ValidArgsFunction: o.controlPlaneFlags.CompleteControlPlane(f),
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.validate())
			util.AssertNilError(o.run())
		},
	}

	o.addFlags(f, cmd)

	return cmd
}
