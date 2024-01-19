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

package openstackcloudprovider

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/utils/openstack/clientconfig"
	ini "gopkg.in/ini.v1"

	unikornv1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"

	"github.com/spjmurray/unikorn-core/pkg/constants"
	"github.com/spjmurray/unikorn-core/pkg/provisioners/application"
	"github.com/spjmurray/unikorn-core/pkg/provisioners/util"

	"sigs.k8s.io/yaml"
)

var (
	// ErrCloudConfiguration is returned when the cloud configuration is not
	// correctly formatted.
	ErrCloudConfiguration = errors.New("invalid cloud configuration")
)

// Provisioner encapsulates control plane provisioning.
type Provisioner struct{}

// New returns a new initialized provisioner object.
func New(getApplication application.GetterFunc) *application.Provisioner {
	provisioner := &Provisioner{}

	return application.New(getApplication).WithGenerator(provisioner).InNamespace("ocp-system")
}

// Ensure the Provisioner interface is implemented.
var _ application.ValuesGenerator = &Provisioner{}

// GenerateCloudConfig does the horrific translation between the myriad ways that OpenStack
// deems necessary to authenticate to the cloud configuration format.  See:
// https://github.com/kubernetes/cloud-provider-openstack/blob/master/docs/openstack-cloud-controller-manager/using-openstack-cloud-controller-manager.md#config-openstack-cloud-controller-manager
//
//nolint:cyclop
func GenerateCloudConfig(cluster *unikornv1.KubernetesCluster) (string, error) {
	var clouds clientconfig.Clouds

	if err := yaml.Unmarshal(*cluster.Spec.Openstack.CloudConfig, &clouds); err != nil {
		return "", err
	}

	cloud, ok := clouds.Clouds[*cluster.Spec.Openstack.Cloud]
	if !ok {
		return "", fmt.Errorf("%w: cloud '%s' not found in clouds.yaml", ErrCloudConfiguration, *cluster.Spec.Openstack.Cloud)
	}

	if cloud.AuthType != clientconfig.AuthV3ApplicationCredential {
		return "", fmt.Errorf("%w: v3applicationcredential auth_type must be specified in clouds.yaml", ErrCloudConfiguration)
	}

	cloudConfig := ini.Empty()

	global, err := cloudConfig.NewSection("Global")
	if err != nil {
		return "", err
	}

	if _, err := global.NewKey("auth-url", cloud.AuthInfo.AuthURL); err != nil {
		return "", err
	}

	if _, err := global.NewKey("application-credential-id", cloud.AuthInfo.ApplicationCredentialID); err != nil {
		return "", err
	}

	if _, err := global.NewKey("application-credential-secret", cloud.AuthInfo.ApplicationCredentialSecret); err != nil {
		return "", err
	}

	loadBalancer, err := cloudConfig.NewSection("LoadBalancer")
	if err != nil {
		return "", err
	}

	if _, err := loadBalancer.NewKey("floating-network-id", *cluster.Spec.Openstack.ExternalNetworkID); err != nil {
		return "", err
	}

	blockStorage, err := cloudConfig.NewSection("BlockStorage")
	if err != nil {
		return "", err
	}

	if _, err := blockStorage.NewKey("ignore-volume-az", "true"); err != nil {
		return "", err
	}

	buffer := &bytes.Buffer{}

	if _, err := cloudConfig.WriteTo(buffer); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

// Generate implements the application.Generator interface.
// Note there is an option, to just pass through the clouds.yaml file, however
// the chart doesn't allow it to be exposed so we need to translate between formats.
func (p *Provisioner) Values(ctx context.Context, _ *string) (interface{}, error) {
	//nolint:forcetypeassert
	cluster := application.FromContext(ctx).(*unikornv1.KubernetesCluster)

	cloudConfig, err := GenerateCloudConfig(cluster)
	if err != nil {
		return nil, err
	}

	cloudConfigHash, err := util.GetConfigurationHash(cloudConfig)
	if err != nil {
		return nil, err
	}

	tolerations := util.ControlPlaneTolerations()
	tolerations = append(tolerations, util.ControlPlaneInitTolerations()...)

	values := map[string]interface{}{
		"commonAnnotations": map[string]interface{}{
			constants.ConfigurationHashAnnotation: cloudConfigHash,
		},
		"cloudConfigContents": cloudConfig,
		"tolerations":         tolerations,
		// See https://github.com/kubernetes/cloud-provider-openstack/issues/2049 for
		// more details, and no-one doing anything about it.
		"controllerExtraArgs": `{{list "--use-service-account-credentials=false" | toYaml}}`,
	}

	return values, nil
}
