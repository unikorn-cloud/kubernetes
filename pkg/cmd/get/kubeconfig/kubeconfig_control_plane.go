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

package kubeconfig

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/generated/clientset/unikorn"
	"github.com/spjmurray/unikorn/pkg/cmd/aliases"
	"github.com/spjmurray/unikorn/pkg/cmd/util"
	"github.com/spjmurray/unikorn/pkg/cmd/util/flags"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/vcluster"

	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type getKubeconfigControlPlaneOptions struct {
	// controlPlaneFlags define control plane scoping.
	controlPlaneFlags flags.ControlPlaneFlags

	// client is the Kubernetes v1 client.
	client kubernetes.Interface

	// unikornClient gives access to our custom resources.
	unikornClient unikorn.Interface
}

// addFlags registers create cluster options flags with the specified cobra command.
func (o *getKubeconfigControlPlaneOptions) addFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.controlPlaneFlags.AddFlags(f, cmd)
}

// complete fills in any options not does automatically by flag parsing.
func (o *getKubeconfigControlPlaneOptions) complete(f cmdutil.Factory, _ []string) error {
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
func (o *getKubeconfigControlPlaneOptions) validate() error {
	return nil
}

// run executes the command.
func (o *getKubeconfigControlPlaneOptions) run() error {
	namespace, err := o.controlPlaneFlags.GetControlPlaneNamespace(context.TODO(), o.unikornClient)
	if err != nil {
		return err
	}

	vc := vcluster.NewClient(o.client)

	kubeconfig, err := vc.Kubeconfig(context.TODO(), namespace, true)
	if err != nil {
		return err
	}

	fmt.Println(kubeconfig)

	return nil
}

// newGetKubeconfigControlPlane creates a command that gets a control plane kubeconfig.
func newGetKubeconfigControlPlane(f cmdutil.Factory) *cobra.Command {
	o := &getKubeconfigControlPlaneOptions{}

	cmd := &cobra.Command{
		Use:     "control-plane",
		Short:   "Get a control plane Kubernetes config",
		Long:    "Get a control plane Kubernetes config",
		Aliases: aliases.ControlPlane,
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.validate())
			util.AssertNilError(o.run())
		},
	}

	o.addFlags(f, cmd)

	return cmd
}
