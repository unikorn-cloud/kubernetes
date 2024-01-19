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
	unikornv1alpha1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/cmd/aliases"
	"github.com/spjmurray/unikorn/pkg/cmd/errors"
	"github.com/spjmurray/unikorn/pkg/cmd/util"
	"github.com/spjmurray/unikorn/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"
)

type createProjectOptions struct {
	// name is the name of the project to create.
	name string

	// client is the Kubernetes v1 client.
	client kubernetes.Interface

	// unikornClient gives access to our custom resources.
	unikornClient unikorn.Interface
}

// complete fills in any options not does automatically by flag parsing.
func (o *createProjectOptions) complete(f cmdutil.Factory, args []string) error {
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

	if len(args) != 1 {
		return errors.ErrIncorrectArgumentNum
	}

	o.name = args[0]

	return nil
}

// validate validates any tainted input not handled by complete() or flags
// processing.
func (o *createProjectOptions) validate() error {
	if len(o.name) == 0 {
		return errors.ErrInvalidName
	}

	return nil
}

// run executes the command.
func (o *createProjectOptions) run() error {
	project := &unikornv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: o.name,
			Labels: map[string]string{
				constants.VersionLabel: constants.Version,
			},
		},
	}

	if _, err := o.unikornClient.UnikornV1alpha1().Projects().Create(context.TODO(), project, metav1.CreateOptions{}); err != nil {
		return err
	}

	fmt.Printf("%s.%s/%s created\n", unikornv1alpha1.ProjectResource, unikornv1alpha1.GroupName, o.name)

	return nil
}

var (
	//nolint:gochecknoglobals
	createProjectLong = templates.LongDesc(`
        Create a project.

	Projects conceptually represent an organization, or a department within an
	organization, and ostensibly map to an OpenStack project.

	Projects map 1:1 to a namespace, and these project namespaces contain
	custom control plane resources.  Thus we can simply off-board users/projects
	with a single delete.

	Projects are cluster scoped and therefore must have globally unique names.`)

	//nolint:gochecknoglobals
	createProjectExample = util.TemplatedExample(`
        # Create a control plane named my-project-name.
        {{.Application}} create project my-project-name`)
)

// newCreateProjectCommand creates a command that can create control planes.
// The initial intention is to have a per-user/organization control plane that
// contains Cluster API in a virtual cluster.
func newCreateProjectCommand(f cmdutil.Factory) *cobra.Command {
	o := &createProjectOptions{}

	cmd := &cobra.Command{
		Use:     "project [flags] my-project-name",
		Short:   "Create a project.",
		Long:    createProjectLong,
		Example: createProjectExample,
		Aliases: aliases.Project,
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.validate())
			util.AssertNilError(o.run())
		},
	}

	return cmd
}
