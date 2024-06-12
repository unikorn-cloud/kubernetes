/*
Copyright 2022-2024 EscherCloud.
Copyright 2024 the Unikorn Authors.

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

package clustermanager

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/unikorn-cloud/core/pkg/constants"
	unikornv1 "github.com/unikorn-cloud/kubernetes/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/kubernetes/pkg/monitor/upgrade/errors"
	"github.com/unikorn-cloud/kubernetes/pkg/monitor/upgrade/util"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Checker struct {
	client client.Client
}

func New(client client.Client) *Checker {
	return &Checker{
		client: client,
	}
}

func (c *Checker) upgradeResource(ctx context.Context, resource *unikornv1.ClusterManager, bundles *unikornv1.ClusterManagerApplicationBundleList, target *unikornv1.ClusterManagerApplicationBundle) error {
	logger := log.FromContext(ctx)

	bundle := bundles.Get(*resource.Spec.ApplicationBundle)
	if bundle == nil {
		return fmt.Errorf("%w: %s", errors.ErrMissingBundle, *resource.Spec.ApplicationBundle)
	}

	// If the current bundle is in preview, then don't offer to upgrade.
	if bundle.Spec.Preview != nil && *bundle.Spec.Preview {
		logger.Info("bundle in preview, ignoring")

		return nil
	}

	// If the current bundle is the best option already, we are done.
	if bundle.Name == target.Name {
		logger.Info("bundle already latest, ignoring")

		return nil
	}

	upgradable := util.UpgradeableResource(resource)

	if resource.Spec.ApplicationBundleAutoUpgrade == nil {
		if bundle.Spec.EndOfLife == nil || time.Now().Before(bundle.Spec.EndOfLife.Time) {
			logger.Info("resource auto-upgrade disabled, ignoring")

			return nil
		}

		logger.Info("resource auto-upgrade disabled, but bundle is end of life, forcing auto-upgrade")

		upgradable = util.NewForcedUpgradeResource(resource)
	}

	// Is it allowed to happen now?  Base it on the UID for ultimate randomness,
	// you can cause a stampede if all the resources are called "default".
	window := util.TimeWindowFromResource(ctx, upgradable)

	if !window.In() {
		logger.Info("not in upgrade window, ignoring", "start", window.Start, "end", window.End)

		return nil
	}

	logger.Info("bundle upgrading", "from", *bundle.Spec.Version, "to", *target.Spec.Version)

	resource.Spec.ApplicationBundle = &target.Name

	return c.client.Update(ctx, resource)
}

func (c *Checker) Check(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("checking for control plane upgrades")

	allBundles := &unikornv1.ClusterManagerApplicationBundleList{}

	if err := c.client.List(ctx, allBundles, &client.ListOptions{}); err != nil {
		return err
	}

	// Extract the potential upgrade target bundles, these are sorted by version, so
	// the newest is on the top, we shall see why later...
	bundles := allBundles.Upgradable()
	if len(bundles.Items) == 0 {
		return errors.ErrNoBundles
	}

	slices.SortStableFunc(bundles.Items, unikornv1.CompareClusterManagerApplicationBundle)

	// Pick the most recent as our upgrade target.
	upgradeTarget := &bundles.Items[len(bundles.Items)-1]

	resources := &unikornv1.ClusterManagerList{}

	if err := c.client.List(ctx, resources, &client.ListOptions{}); err != nil {
		return err
	}

	for i := range resources.Items {
		resource := &resources.Items[i]

		logger := logger.WithValues(
			"organization", resource.Labels[constants.OrganizationLabel],
			"project", resource.Labels[constants.ProjectLabel],
			"clustermanager", resource.Name,
		)

		if err := c.upgradeResource(log.IntoContext(ctx, logger), resource, allBundles, upgradeTarget); err != nil {
			return err
		}
	}

	return nil
}
