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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/generated/clientset/unikorn"
	unikornv1alpha1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/cmd/aliases"
	"github.com/spjmurray/unikorn/pkg/cmd/errors"
	"github.com/spjmurray/unikorn/pkg/cmd/util"
	"github.com/spjmurray/unikorn/pkg/cmd/util/flags"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type getControlPlaneOptions struct {
	// projectFlags defines project scoping.
	projectFlags flags.ProjectFlags

	// name allows explicit filtering of control plane namespaces.
	names []string

	// getPrintFlags is a generic and reduced set of printing options.
	getPrintFlags *getPrintFlags

	// f is the factory used to create clients.
	f cmdutil.Factory

	// client gives access to our custom resources.
	client unikorn.Interface
}

// newGetControlPlaneOptions returns a correctly initialized set of options.
func newGetControlPlaneOptions() *getControlPlaneOptions {
	return &getControlPlaneOptions{
		getPrintFlags: newGetPrintFlags(),
	}
}

// addFlags registers create cluster options flags with the specified cobra command.
func (o *getControlPlaneOptions) addFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.projectFlags.AddFlags(f, cmd)
	o.getPrintFlags.addFlags(cmd)
}

// complete fills in any options not does automatically by flag parsing.
func (o *getControlPlaneOptions) complete(f cmdutil.Factory, args []string) error {
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

// validate validates any tainted input not handled by complete() or flags
// processing.
func (o *getControlPlaneOptions) validate() error {
	if o.names != nil {
		for _, name := range o.names {
			if len(name) == 0 {
				return fmt.Errorf(`%w: "%s"`, errors.ErrInvalidName, name)
			}
		}
	}

	if len(o.projectFlags.Project) == 0 {
		return fmt.Errorf(`%w: "%s"`, errors.ErrInvalidName, o.projectFlags.Project)
	}

	return nil
}

// run executes the command.
func (o *getControlPlaneOptions) run() error {
	namespace, err := o.projectFlags.GetProjectNamespace(context.TODO(), o.client)
	if err != nil {
		return err
	}

	// We are using the "kubectl get" library to retrieve resources.  That command
	// is generic, it accepts a kind and name(s), or a list of type/name tuples.
	// In our case, the type is implicit, so we need to prepend it to keep things
	// working as they should.
	args := []string{unikornv1alpha1.ControlPlaneResource}
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

// newGetControlPlaneCommand returns a command that is able to get or list Cluster API
// control planes found on the management cluster.
func newGetControlPlaneCommand(f cmdutil.Factory) *cobra.Command {
	o := newGetControlPlaneOptions()

	cmd := &cobra.Command{
		Use:               "control-plane",
		Short:             "Get or list Cluster API control planes",
		Long:              "Get or list Cluster API control planes",
		Aliases:           aliases.ControlPlane,
		ValidArgsFunction: o.projectFlags.CompleteControlPlane(f),
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.validate())
			util.AssertNilError(o.run())
		},
	}

	o.addFlags(f, cmd)

	return cmd
}
