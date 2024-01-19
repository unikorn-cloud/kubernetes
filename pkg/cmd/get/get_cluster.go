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
	"context"

	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/generated/clientset/unikorn"
	unikornv1alpha1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/cmd/aliases"
	"github.com/spjmurray/unikorn/pkg/cmd/util"
	"github.com/spjmurray/unikorn/pkg/cmd/util/flags"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// getClusterOptions defines a set of options that are required to get
// a cluster.
type getClusterOptions struct {
	// controlPlaneFlags define control plane scoping.
	controlPlaneFlags flags.ControlPlaneFlags

	// names is an explicit set of resource names to get.
	names []string

	// getPrintFlags is a generic and reduced set of printing options.
	getPrintFlags *getPrintFlags

	// f is the factory used to create clients.
	f cmdutil.Factory

	// client gives access to our custom resources.
	client unikorn.Interface
}

func newGetClusterOptions() *getClusterOptions {
	return &getClusterOptions{
		getPrintFlags: newGetPrintFlags(),
	}
}

// addFlags registers get cluster options flags with the specified cobra command.
func (o *getClusterOptions) addFlags(cmd *cobra.Command, f cmdutil.Factory) {
	o.controlPlaneFlags.AddFlags(f, cmd)
	o.getPrintFlags.addFlags(cmd)
}

// complete fills in any options not does automatically by flag parsing.
func (o *getClusterOptions) complete(f cmdutil.Factory, args []string) error {
	o.f = f

	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}

	if o.client, err = unikorn.NewForConfig(config); err != nil {
		return err
	}

	if len(args) != 0 {
		o.names = util.UniqueString(args)
	}

	return nil
}

// run executes the command.
func (o *getClusterOptions) run() error {
	namespace, err := o.controlPlaneFlags.GetControlPlaneNamespace(context.TODO(), o.client)
	if err != nil {
		return err
	}

	// We are using the "kubectl get" library to retrieve resources.  That command
	// is generic, it accepts a kind and name(s), or a list of type/name tuples.
	// In our case, the type is implicit, so we need to prepend it to keep things
	// working as they should.
	args := []string{unikornv1alpha1.KubernetesClusterResource}
	args = append(args, o.names...)

	r := o.f.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).
		ResourceTypeOrNameArgs(true, args...).
		ContinueOnError().
		Latest().
		Flatten().
		TransformRequests(o.getPrintFlags.transformRequests).
		Do()

	if err := r.Err(); err != nil {
		return err
	}

	if err := o.getPrintFlags.printResult(r); err != nil {
		return err
	}

	return nil
}

var (
	//nolint:gochecknoglobals
	getClusterExamples = util.TemplatedExample(`
        # List Kubernetes clusters in control plane foo
        {{.Application}} get cluster --project foo --control-plane bar`)
)

// newGetClusterCommand returns a command that is able to get or list Kubernetes clusters
// found in the provided Cluster API control plane.
func newGetClusterCommand(f cmdutil.Factory) *cobra.Command {
	o := newGetClusterOptions()

	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Get or list Kubernetes clusters",
		Long:              "Get or list Kubernetes clusters",
		Example:           getClusterExamples,
		Aliases:           aliases.Cluster,
		ValidArgsFunction: o.controlPlaneFlags.CompleteCluster(f),
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.run())
		},
	}

	o.addFlags(cmd, f)

	return cmd
}
