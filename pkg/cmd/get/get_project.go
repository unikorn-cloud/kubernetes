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
	"fmt"

	"github.com/spf13/cobra"

	unikornv1alpha1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/cmd/aliases"
	"github.com/spjmurray/unikorn/pkg/cmd/errors"
	"github.com/spjmurray/unikorn/pkg/cmd/util"

	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/completion"
)

type getProjectOptions struct {
	// name allows explicit filtering of control plane namespaces.
	names []string

	// getPrintFlags is a generic and reduced set of printing options.
	getPrintFlags *getPrintFlags

	// f is the factory used to create clients.
	f cmdutil.Factory

	// client is the Kubernetes v1 client.
	client kubernetes.Interface
}

// newGetProjectOptions returns a correctly initialized set of options.
func newGetProjectOptions() *getProjectOptions {
	return &getProjectOptions{
		getPrintFlags: newGetPrintFlags(),
	}
}

func (o *getProjectOptions) addFlags(cmd *cobra.Command) {
	o.getPrintFlags.addFlags(cmd)
}

// complete fills in any options not does automatically by flag parsing.
func (o *getProjectOptions) complete(f cmdutil.Factory, args []string) error {
	o.f = f

	var err error

	if o.client, err = f.KubernetesClientSet(); err != nil {
		return err
	}

	if len(args) != 0 {
		o.names = util.UniqueString(args)
	}

	return nil
}

// validate validates any tainted input not handled by complete() or flags
// processing.
func (o *getProjectOptions) validate() error {
	if o.names != nil {
		for _, name := range o.names {
			if len(name) == 0 {
				return fmt.Errorf(`%w: "%s"`, errors.ErrInvalidName, name)
			}
		}
	}

	return nil
}

// run executes the command.
func (o *getProjectOptions) run() error {
	// We are using the "kubectl get" library to retrieve resources.  That command
	// is generic, it accepts a kind and name(s), or a list of type/name tuples.
	// In our case, the type is implicit, so we need to prepend it to keep things
	// working as they should.
	args := []string{unikornv1alpha1.ProjectResource}
	args = append(args, o.names...)

	r := o.f.NewBuilder().
		Unstructured().
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
	getProjectExample = util.TemplatedExample(`
	# Get all projects.
	{{.Application}} get project

	# Get a single project named my-project-name.
	{{.Application}} get project my-project-name

	# Get multiple projects.
	{{.Application}} get project my-project-name my-other-project-name

	# Get all projects formatted in YAML.
	{{.Application}} get project -o yaml`)
)

// newGetProjectCommand returns a command that is able to get or list Cluster API
// control planes found on the management cluster.
func newGetProjectCommand(f cmdutil.Factory) *cobra.Command {
	o := newGetProjectOptions()

	cmd := &cobra.Command{
		Use:               "project",
		Short:             "Get or list projects",
		Long:              "Get or list projects",
		Example:           getProjectExample,
		Aliases:           aliases.Project,
		ValidArgsFunction: completion.ResourceNameCompletionFunc(f, unikornv1alpha1.ProjectResource),
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.validate())
			util.AssertNilError(o.run())
		},
	}

	o.addFlags(cmd)

	return cmd
}
