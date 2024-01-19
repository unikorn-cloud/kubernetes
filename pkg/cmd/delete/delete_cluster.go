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

package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/generated/clientset/unikorn"
	unikornv1alpha1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/cmd/aliases"
	"github.com/spjmurray/unikorn/pkg/cmd/errors"
	"github.com/spjmurray/unikorn/pkg/cmd/util"
	"github.com/spjmurray/unikorn/pkg/cmd/util/flags"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// deleteClusterOptions defines a set of options that are required to delete
// a cluster.
type deleteClusterOptions struct {
	// controlPlaneFlags define control plane scoping.
	controlPlaneFlags flags.ControlPlaneFlags

	// deleteFlags define common deletion options.
	deleteFlags flags.DeleteFlags

	// names are the names of the clusters.
	names []string

	// client gives access to our custom resources.
	client unikorn.Interface
}

// addFlags registers delete cluster options flags with the specified cobra command.
func (o *deleteClusterOptions) addFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.controlPlaneFlags.AddFlags(f, cmd)
	o.deleteFlags.AddFlags(f, cmd)
}

// completeNames either sets the names explcitly via the CLI or implicitly if --all
// is specified.
func (o *deleteClusterOptions) completeNames(args []string) error {
	if !o.deleteFlags.All {
		o.names = util.UniqueString(args)

		return nil
	}

	namespace, err := o.controlPlaneFlags.GetControlPlaneNamespace(context.TODO(), o.client)
	if err != nil {
		return err
	}

	resources, err := o.client.UnikornV1alpha1().KubernetesClusters(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	o.names = make([]string, len(resources.Items))

	for i, resource := range resources.Items {
		o.names[i] = resource.Name
	}

	return nil
}

// complete fills in any options not does automatically by flag parsing.
func (o *deleteClusterOptions) complete(f cmdutil.Factory, args []string) error {
	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}

	if o.client, err = unikorn.NewForConfig(config); err != nil {
		return err
	}

	if err := o.completeNames(args); err != nil {
		return err
	}

	return nil
}

// validate validates any tainted input not handled by complete() or flags
// processing.
func (o *deleteClusterOptions) validate() error {
	if !o.deleteFlags.All && len(o.names) == 0 {
		return fmt.Errorf(`%w: resource names or --all must be specified`, errors.ErrInvalidName)
	}

	return nil
}

// run executes the command.
func (o *deleteClusterOptions) run() error {
	namespace, err := o.controlPlaneFlags.GetControlPlaneNamespace(context.TODO(), o.client)
	if err != nil {
		return err
	}

	for _, name := range o.names {
		if err := o.client.UnikornV1alpha1().KubernetesClusters(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
			return err
		}

		fmt.Printf("%s.%s/%s deleted\n", unikornv1alpha1.KubernetesClusterResource, unikornv1alpha1.GroupName, name)
	}

	return nil
}

var (
	//nolint:gochecknoglobals
	deleteClusterExamples = util.TemplatedExample(`
        # Delete a Kubernetes cluster
        {{.Application}} delete cluster --control-plane foo`)
)

// newDeleteClusterCommand creates a command that deletes a Kubenretes cluster in the
// specified Cluster API control plane.
func newDeleteClusterCommand(f cmdutil.Factory) *cobra.Command {
	o := &deleteClusterOptions{}

	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Delete a Kubernetes cluster",
		Long:              "Delete a Kubernetes cluster",
		Example:           deleteClusterExamples,
		Aliases:           aliases.Cluster,
		ValidArgsFunction: o.controlPlaneFlags.CompleteCluster(f),
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.validate())
			util.AssertNilError(o.run())
		},
	}

	o.addFlags(f, cmd)

	return cmd
}
