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

package create

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/generated/clientset/unikorn"
	unikornv1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/cmd/aliases"
	"github.com/spjmurray/unikorn/pkg/cmd/errors"
	"github.com/spjmurray/unikorn/pkg/cmd/util"
	"github.com/spjmurray/unikorn/pkg/cmd/util/flags"
	"github.com/spjmurray/unikorn/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"
)

type createControlPlaneOptions struct {
	// projectFlags defines project scoping.
	projectFlags flags.ProjectFlags

	// name is the name of the control plane to create.
	name string

	// applicationBundle is the version to provision.
	applicationBundle string

	// client gives access to our custom resources.
	client unikorn.Interface
}

// addFlags registers create cluster options flags with the specified cobra command.
func (o *createControlPlaneOptions) addFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.projectFlags.AddFlags(f, cmd)
	flags.RequiredStringVarWithCompletion(cmd, &o.applicationBundle, "application-bundle", "", "Application bundle, defining component versions, to deploy", flags.CompleteControlPlaneApplicationBundle(f))
}

// complete fills in any options not does automatically by flag parsing.
func (o *createControlPlaneOptions) complete(f cmdutil.Factory, args []string) error {
	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}

	if o.client, err = unikorn.NewForConfig(config); err != nil {
		return err
	}

	if len(args) != 1 {
		return errors.ErrIncorrectArgumentNum
	}

	o.name = args[0]

	return nil
}

// validate validates any tainted input not handled by complete() or flags
// processing.
func (o *createControlPlaneOptions) validate() error {
	if len(o.name) == 0 {
		return errors.ErrInvalidName
	}

	return nil
}

// run executes the command.
func (o *createControlPlaneOptions) run() error {
	namespace, err := o.projectFlags.GetProjectNamespace(context.TODO(), o.client)
	if err != nil {
		return err
	}

	controlPlane := &unikornv1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name: o.name,
			Labels: map[string]string{
				constants.VersionLabel: constants.Version,
				constants.ProjectLabel: o.projectFlags.Project,
			},
		},
		Spec: unikornv1.ControlPlaneSpec{
			ApplicationBundle: &o.applicationBundle,
		},
	}

	if _, err := o.client.UnikornV1alpha1().ControlPlanes(namespace).Create(context.TODO(), controlPlane, metav1.CreateOptions{}); err != nil {
		return err
	}

	fmt.Printf("%s.%s/%s created\n", unikornv1.ControlPlaneResource, unikornv1.GroupName, o.name)

	return nil
}

var (
	//nolint:gochecknoglobals
	createControlPlaneLong = templates.LongDesc(`
        Create a Cluster API control plane.

        Control planes are modelled on Kubernetes namespaces, this gives
        us a primitive to label, and annotate, to aid in life-cycle management.

        Each control plane namespace will contain an instance of a loft.io
        vcluster.  The use of vclusters allows a level of isolation between
        users in a multi-tenancy environment.  It also allows trivial deletion
        of resources contained within that vcluster as that is not subject
        to finalizers.`)

	//nolint:gochecknoglobals
	createControlPlaneExample = util.TemplatedExample(`
        # Create a control plane named my-control-plane-name.
        {{.Application}} create control-plane my-control-plane-name`)
)

// newCreateControlPlaneCommand creates a command that can create control planes.
// The initial intention is to have a per-user/organization control plane that
// contains Cluster API in a virtual cluster.
func newCreateControlPlaneCommand(f cmdutil.Factory) *cobra.Command {
	o := &createControlPlaneOptions{}

	cmd := &cobra.Command{
		Use:     "control-plane [flags] my-control-plane-name",
		Short:   "Create a Cluster API control plane.",
		Long:    createControlPlaneLong,
		Example: createControlPlaneExample,
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
