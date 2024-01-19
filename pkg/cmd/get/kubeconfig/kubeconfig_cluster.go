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
	"github.com/spjmurray/unikorn/pkg/cmd/errors"
	"github.com/spjmurray/unikorn/pkg/cmd/util"
	"github.com/spjmurray/unikorn/pkg/cmd/util/flags"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/clusteropenstack"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/vcluster"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type getKubeconfigClusterOptions struct {
	// clusterFlags define cluster scoping.
	clusterFlags flags.ClusterFlags

	// client is the Kubernetes v1 client.
	client kubernetes.Interface

	// unikornClient gives access to our custom resources.
	unikornClient unikorn.Interface
}

// addFlags registers create cluster options flags with the specified cobra command.
func (o *getKubeconfigClusterOptions) addFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.clusterFlags.AddFlags(f, cmd)
}

// complete fills in any options not does automatically by flag parsing.
func (o *getKubeconfigClusterOptions) complete(f cmdutil.Factory, _ []string) error {
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
func (o *getKubeconfigClusterOptions) validate() error {
	return nil
}

// run executes the command.
func (o *getKubeconfigClusterOptions) run() error {
	namespace, err := o.clusterFlags.GetControlPlaneNamespace(context.TODO(), o.unikornClient)
	if err != nil {
		return err
	}

	vc := vcluster.NewClient(o.client)

	vclusterConfig, err := vc.RESTConfig(context.TODO(), namespace, true)
	if err != nil {
		return err
	}

	vclusterClient, err := kubernetes.NewForConfig(vclusterConfig)
	if err != nil {
		return err
	}

	cluster, err := o.unikornClient.UnikornV1alpha1().KubernetesClusters(namespace).Get(context.TODO(), o.clusterFlags.Cluster, metav1.GetOptions{})
	if err != nil {
		return err
	}

	secret, err := vclusterClient.CoreV1().Secrets(o.clusterFlags.Cluster).Get(context.TODO(), clusteropenstack.KubeconfigSecretName(cluster), metav1.GetOptions{})
	if err != nil {
		return err
	}

	kubeconfig, ok := secret.Data["value"]
	if !ok {
		return fmt.Errorf("%w: kubeconfig value missing", errors.ErrNotFound)
	}

	fmt.Println(string(kubeconfig))

	return nil
}

// newGetKubeconfigCluster creates a command that gets a cluster kubeconfig.
func newGetKubeconfigCluster(f cmdutil.Factory) *cobra.Command {
	o := &getKubeconfigClusterOptions{
		clusterFlags: flags.ClusterFlags{
			ClusterRequired: true,
		},
	}

	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "Get a cluster Kubernetes config",
		Long:    "Get a cluster Kubernetes config",
		Aliases: aliases.Cluster,
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.validate())
			util.AssertNilError(o.run())
		},
	}

	o.addFlags(f, cmd)

	return cmd
}
