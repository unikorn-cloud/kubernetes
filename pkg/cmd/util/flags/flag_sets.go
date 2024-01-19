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

package flags

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/generated/clientset/unikorn"
	unikornv1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	computil "k8s.io/kubectl/pkg/util/completion"
)

var (
	// ErrUnavailable is for when the resource status reports unready.
	ErrUnavailable = errors.New("resource unavailable")

	// ErrNamespace is for when the resource status doesn't contain a namespace.
	ErrNamespace = errors.New("namespace error")
)

// getClient returns a unikorn client from a factory.
func getClient(f cmdutil.Factory) (unikorn.Interface, error) {
	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	client, err := unikorn.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CompleteControlPlaneApplicationBundle provides tab completion for application bundles.
func CompleteControlPlaneApplicationBundle(f cmdutil.Factory) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := getClient(f)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		applicationBundles, err := client.UnikornV1alpha1().ControlPlaneApplicationBundles().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, b := range applicationBundles.Items {
			matches = append(matches, b.Name)
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleteKubernetesClusterApplicationBundle provides tab completion for application bundles.
func CompleteKubernetesClusterApplicationBundle(f cmdutil.Factory) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := getClient(f)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		applicationBundles, err := client.UnikornV1alpha1().KubernetesClusterApplicationBundles().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, b := range applicationBundles.Items {
			matches = append(matches, b.Name)
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleteProject provides tab completion for the specified resource type.
func CompleteProject(f cmdutil.Factory) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return computil.ResourceNameCompletionFunc(f, unikornv1.ProjectResource)
}

// ProjectFlags are required flags for a project scoped resource.
type ProjectFlags struct {
	// project defines the project a resource is under.
	Project string
}

// AddFlags adds the flags to a cobra command.
func (o *ProjectFlags) AddFlags(f cmdutil.Factory, cmd *cobra.Command) {
	RequiredStringVarWithCompletion(cmd, &o.Project, "project", "", "Project scope of a resource.", CompleteProject(f))
}

// CompleteControlPlane provides tab completion for the specified resource type.
//
//nolint:dupl
func (o *ProjectFlags) CompleteControlPlane(f cmdutil.Factory) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := getClient(f)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		namespace, err := o.GetProjectNamespace(context.TODO(), client)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		controlPlanes, err := client.UnikornV1alpha1().ControlPlanes(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, cp := range controlPlanes.Items {
			if strings.HasPrefix(cp.Name, toComplete) {
				matches = append(matches, cp.Name)
			}
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// GetProjectNamespace figures out the namespace associated with a project.
func (o *ProjectFlags) GetProjectNamespace(ctx context.Context, client unikorn.Interface) (string, error) {
	p, err := client.UnikornV1alpha1().Projects().Get(ctx, o.Project, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	namespace := p.Status.Namespace

	if namespace == "" {
		return "", fmt.Errorf("%w: project namespace unset", ErrNamespace)
	}

	return namespace, nil
}

// ControlPlaneFlags are required flags for a project scoped resource.
type ControlPlaneFlags struct {
	// ProjectFlags define the project for a control plane.
	ProjectFlags

	// ControlPlane defines the control plane a resource is under.
	ControlPlane string
}

// AddFlags adds the flags to a cobra command.
func (o *ControlPlaneFlags) AddFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.ProjectFlags.AddFlags(f, cmd)

	RequiredStringVarWithCompletion(cmd, &o.ControlPlane, "control-plane", "", "Control plane scope of a resource.", o.CompleteControlPlane(f))
}

// CompleteCluster provides tab completion for the specified resource type.
//
//nolint:dupl
func (o *ControlPlaneFlags) CompleteCluster(f cmdutil.Factory) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := getClient(f)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		namespace, err := o.GetControlPlaneNamespace(context.TODO(), client)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		clusters, err := client.UnikornV1alpha1().KubernetesClusters(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, cluster := range clusters.Items {
			if strings.HasPrefix(cluster.Name, toComplete) {
				matches = append(matches, cluster.Name)
			}
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// GetControlPlaneNamespace figures out the namespace associated with a project's control plane.
func (o *ControlPlaneFlags) GetControlPlaneNamespace(ctx context.Context, client unikorn.Interface) (string, error) {
	namespace, err := o.GetProjectNamespace(ctx, client)
	if err != nil {
		return "", err
	}

	cp, err := client.UnikornV1alpha1().ControlPlanes(namespace).Get(ctx, o.ControlPlane, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	namespace = cp.Status.Namespace

	if namespace == "" {
		return "", fmt.Errorf("%w: control plane namespace unset", ErrNamespace)
	}

	return namespace, nil
}

// ClusterFlags are flags for a cluster scoped resource.  Unlike projects
// and control planes there is a 1:* cardinality so they aren't required.
type ClusterFlags struct {
	// ControlPlaneFlags define the project and control plane for a cluster.
	ControlPlaneFlags

	// Cluster defines the cluster a resource belongs to.
	Cluster string

	// Indicates whether the cluster parameter is required by the CLI.
	ClusterRequired bool
}

// AddFlags adds the flags to a cobra command.
func (o *ClusterFlags) AddFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.ControlPlaneFlags.AddFlags(f, cmd)

	registerFunc := StringVarWithCompletion

	if o.ClusterRequired {
		registerFunc = RequiredStringVarWithCompletion
	}

	// Note: cannot use "cluster" here as it clashes with cli-runtime.
	registerFunc(cmd, &o.Cluster, "kubernetes-cluster", "", "Cluster scope of a resource.", o.CompleteCluster(f))
}

// DeleteFlags define common deletion options.
type DeleteFlags struct {
	// All defines whether to delete all resources.
	All bool
}

// AddFlags adds the flags to a cobra command.
func (o *DeleteFlags) AddFlags(_ cmdutil.Factory, cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.All, "all", false, "Select all resources that match the query.")
}
