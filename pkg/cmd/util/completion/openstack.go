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

package completion

import (
	"context"
	"strings"

	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/pkg/providers/openstack"
)

// CloudCompletionFunc parses clouds.yaml and supplies matching cloud names.
func CloudCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	clouds, err := clientconfig.LoadCloudsYAML()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var matches []string

	for name := range clouds {
		if strings.HasPrefix(name, toComplete) {
			matches = append(matches, name)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

// OpenstackExternalNetworkCompletionFunc lists any matching external networks by ID.
// Yes this isn't particularly human friendly, but the ID is the only unique identifier.
// Names can alias which makes mapping from name to ID practically useless.
func OpenstackExternalNetworkCompletionFunc(cloud *string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := openstack.NewNetworkClient(openstack.NewCloudsProvider(*cloud))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		results, err := client.ExternalNetworks(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, network := range results {
			if strings.HasPrefix(network.ID, toComplete) {
				matches = append(matches, network.ID)
			}
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// OpenstackSSHKeyCompletionFunc lists any matching ssh key pairs by name.
func OpenstackSSHKeyCompletionFunc(cloud *string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := openstack.NewComputeClient(&openstack.ComputeOptions{}, openstack.NewCloudsProvider(*cloud))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		results, err := client.KeyPairs(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, keypair := range results {
			// TODO: there is a Type ("ssh") field, but it seems this library
			// is too old.
			if strings.HasPrefix(keypair.Name, toComplete) {
				matches = append(matches, keypair.Name)
			}
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// OpenstackFlavorCompletionFunc lists any matching flavors by name.
func OpenstackFlavorCompletionFunc(cloud *string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := openstack.NewComputeClient(&openstack.ComputeOptions{}, openstack.NewCloudsProvider(*cloud))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		results, err := client.Flavors(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, flavor := range results {
			if strings.HasPrefix(flavor.Flavor.Name, toComplete) {
				matches = append(matches, flavor.Name)
			}
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// OpenstackImageCompletionFunc lists any matching images by name.
func OpenstackImageCompletionFunc(cloud *string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := openstack.NewImageClient(openstack.NewCloudsProvider(*cloud))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		results, err := client.Images(context.Background(), nil, nil)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, image := range results {
			if strings.HasPrefix(image.Name, toComplete) {
				matches = append(matches, image.Name)
			}
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// OpenstackComputeAvailabilityZoneCompletionFunc lists any matching availability zones by name.
func OpenstackComputeAvailabilityZoneCompletionFunc(cloud *string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := openstack.NewComputeClient(&openstack.ComputeOptions{}, openstack.NewCloudsProvider(*cloud))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		results, err := client.AvailabilityZones(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, availabilityZone := range results {
			if strings.HasPrefix(availabilityZone.ZoneName, toComplete) {
				matches = append(matches, availabilityZone.ZoneName)
			}
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// OpenstackVolumeAvailabilityZoneCompletionFunc lists any matching availability zones by name.
func OpenstackVolumeAvailabilityZoneCompletionFunc(cloud *string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := openstack.NewBlockStorageClient(openstack.NewCloudsProvider(*cloud))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		results, err := client.AvailabilityZones(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string

		for _, availabilityZone := range results {
			if strings.HasPrefix(availabilityZone.ZoneName, toComplete) {
				matches = append(matches, availabilityZone.ZoneName)
			}
		}

		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}
