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
	"net"

	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/spf13/cobra"

	"github.com/spjmurray/unikorn/generated/clientset/unikorn"
	unikornv1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/cmd/aliases"
	"github.com/spjmurray/unikorn/pkg/cmd/errors"
	"github.com/spjmurray/unikorn/pkg/cmd/util"
	"github.com/spjmurray/unikorn/pkg/cmd/util/completion"
	"github.com/spjmurray/unikorn/pkg/cmd/util/flags"
	"github.com/spjmurray/unikorn/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"

	"sigs.k8s.io/yaml"
)

const (
	defaultControlPlaneReplicas = 3
)

var (
	//nolint:gochecknoglobals
	defaultNodeNetwork = net.IPNet{
		IP:   net.IPv4(192, 168, 0, 0),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	}

	//nolint:gochecknoglobals
	defaultPodNetwork = net.IPNet{
		IP:   net.IPv4(10, 0, 0, 0),
		Mask: net.IPv4Mask(255, 0, 0, 0),
	}

	//nolint:gochecknoglobals
	defaultServiceNetwork = net.IPNet{
		IP:   net.IPv4(172, 16, 0, 0),
		Mask: net.IPv4Mask(255, 240, 0, 0),
	}

	//nolint:gochecknoglobals
	defaultDNSNameservers = []net.IP{
		net.IPv4(8, 8, 8, 8),
	}
)

// createClusterOptions defines a set of options that are required to create
// a cluster.
type createClusterOptions struct {
	// controlPlaneFlags define control plane scoping.
	controlPlaneFlags flags.ControlPlaneFlags

	// name is the name of the cluster.
	name string

	// applicationBundle is the version to provision.
	applicationBundle string

	// cloud indicates the clouds.yaml key to use.  If only one exists it
	// will default to that, otherwise it's a required parameter.
	cloud string

	// clouds is set during completion, and is a filtered version containing
	// only the specified cloud.
	clouds []byte

	// caCert is derived from clouds during completion.
	caCert []byte

	// version defines the Kubernetes version to install.
	version flags.SemverFlag

	// externalNetworkID is an internet facing Openstack network to provision
	// VIPs on for load balancers and stuff.
	externalNetworkID string

	// nodeNetwork is the network prefix Kubernetes nodes live on.
	nodeNetwork net.IPNet

	// podNetwork is the network prefix Kubernetes pods live on.
	podNetwork net.IPNet

	// service Network is the network prefix Kubernetes services live on.
	serviceNetwork net.IPNet

	// dnsNameservers is a list of nameservers for pods and nodes to use.
	dnsNameservers []net.IP

	// image defines the Openstack image for Kubernetes nodes.
	image string

	// flavor defines the Openstack VM flavor Kubernetes control
	// planes use.
	flavor string

	// replicas defines the number of replicas (nodes) for
	// Kubernetes control planes.
	replicas int

	// diskSize defines the persistent volume size to provision with.
	diskSize flags.QuantityFlag

	// computeAvailabilityZone defines in what Openstack failure domain the Kubernetes
	// cluster will be provisioned in.
	computeAvailabilityZone string

	// volumeAvailabilityZone defines in what Openstack failure domain volumes
	// will be provisoned in.
	volumeAvailabilityZone string

	// sshKeyName defines the SSH key to inject onto the Kubernetes cluster.
	sshKeyName string

	// autoscaling allows the cluster to determine its own destiny, not the
	// user.
	autoscaling bool

	// ingress allows the cluster to be provisioned with an ingress controller.
	ingress bool

	// SANs allows the Kuberenetes API to generate a set of X.509 SANs
	// in its certificate.
	SANs []string

	// allowedPrefixes allows the Kubernetes API firewall.
	allowedPrefixes flags.IPNetSliceFlag

	// client gives access to our custom resources.
	client unikorn.Interface
}

// addFlags registers create cluster options flags with the specified cobra command.
func (o *createClusterOptions) addFlags(f cmdutil.Factory, cmd *cobra.Command) {
	o.controlPlaneFlags.AddFlags(f, cmd)

	// Unikorn options.
	flags.RequiredStringVarWithCompletion(cmd, &o.applicationBundle, "application-bundle", "", "Application bundle, defining component versions, to deploy", flags.CompleteKubernetesClusterApplicationBundle(f))

	// Openstack configuration options.
	flags.RequiredStringVarWithCompletion(cmd, &o.cloud, "cloud", "", "Cloud config to use within clouds.yaml.", completion.CloudCompletionFunc)

	// Kubernetes options.
	flags.RequiredVar(cmd, &o.version, "version", "Kubernetes version to deploy.  Provisioning will be faster if this matches the version preloaded on images defined by the --control-plane-image and --workload-image flags.")

	// Networking options.
	flags.RequiredStringVarWithCompletion(cmd, &o.externalNetworkID, "external-network", "", "Openstack external network (see: 'openstack network list --external'.)", completion.OpenstackExternalNetworkCompletionFunc(&o.cloud))
	cmd.Flags().IPNetVar(&o.nodeNetwork, "node-network", defaultNodeNetwork, "Node network prefix.")
	cmd.Flags().IPNetVar(&o.podNetwork, "pod-network", defaultPodNetwork, "Pod network prefix.")
	cmd.Flags().IPNetVar(&o.serviceNetwork, "service-network", defaultServiceNetwork, "Service network prefix.")
	cmd.Flags().IPSliceVar(&o.dnsNameservers, "dns-nameservers", defaultDNSNameservers, "DNS nameservers for pods. (format: 1.1.1.1,8.8.8.8)")
	cmd.Flags().StringSliceVar(&o.SANs, "api-sans", nil, "Specifies X.509 subject alternative names to generate in the API certificate. (format: foo.acme.com,bar.acme.com)")
	cmd.Flags().Var(&o.allowedPrefixes, "api-allowed-prefixes", "Specifies network prefixs allowed to use the Kubernetes API. (format: 1.1.1.1/32,2.2.2.2/32)")

	// Kubernetes control plane options.
	flags.RequiredStringVarWithCompletion(cmd, &o.flavor, "flavor", "", "Kubernetes control plane Openstack flavor (see: 'openstack flavor list'.)", completion.OpenstackFlavorCompletionFunc(&o.cloud))
	cmd.Flags().IntVar(&o.replicas, "replicas", defaultControlPlaneReplicas, "Kubernetes control plane replicas.")
	cmd.Flags().Var(&o.diskSize, "disk-size", "Kubernetes control plane disk size, defaults to that of the machine flavor.")

	// Openstack provisioning options.
	flags.RequiredStringVarWithCompletion(cmd, &o.image, "image", "", "Kubernetes Openstack image (see: 'openstack image list'.)", completion.OpenstackImageCompletionFunc(&o.cloud))
	flags.RequiredStringVarWithCompletion(cmd, &o.computeAvailabilityZone, "compute-availability-zone", "", "Openstack availability zone to provision into.  Only one is supported currently (see: 'openstack availability zone list --compute'.)", completion.OpenstackComputeAvailabilityZoneCompletionFunc(&o.cloud))
	flags.RequiredStringVarWithCompletion(cmd, &o.volumeAvailabilityZone, "volume-availability-zone", "", "Openstack availability zone to provision into.  Only one is supported currently (see: 'openstack availability zone list --volume'.)", completion.OpenstackVolumeAvailabilityZoneCompletionFunc(&o.cloud))
	flags.StringVarWithCompletion(cmd, &o.sshKeyName, "ssh-key-name", "", "Openstack SSH key to inject onto the Kubernetes nodes (see: 'openstack keypair list'.)", completion.OpenstackSSHKeyCompletionFunc(&o.cloud))

	// Feature enablement.
	cmd.Flags().BoolVar(&o.autoscaling, "enable-autoscaling", false, "Enables cluster auto-scaling. To function, you must configure autoscaling on individual workload pools.")
	cmd.Flags().BoolVar(&o.ingress, "enable-ingress", false, "Enables an ingress controller.")
}

// complete fills in any options not does automatically by flag parsing.
func (o *createClusterOptions) complete(f cmdutil.Factory, args []string) error {
	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}

	if o.client, err = unikorn.NewForConfig(config); err != nil {
		return err
	}

	if err := o.completeOpenstackConfig(); err != nil {
		return err
	}

	if len(args) != 1 {
		return errors.ErrIncorrectArgumentNum
	}

	o.name = args[0]

	return nil
}

// completeOpenstackConfig loads the Openstack configuration and derives some options
// from that file.
func (o *createClusterOptions) completeOpenstackConfig() error {
	clouds, err := clientconfig.LoadCloudsYAML()
	if err != nil {
		return err
	}

	// Ensure the cloud exists.
	cloud, ok := clouds[o.cloud]
	if !ok {
		return fmt.Errorf("%w: cloud '%s' not found in clouds.yaml", errors.ErrNotFound, o.cloud)
	}

	// Build the fitered clouds.yaml for use by the provisioner.
	filteredClouds := &clientconfig.Clouds{
		Clouds: map[string]clientconfig.Cloud{
			o.cloud: cloud,
		},
	}

	filteredCloudsYaml, err := yaml.Marshal(filteredClouds)
	if err != nil {
		return err
	}

	o.clouds = filteredCloudsYaml

	return nil
}

// validate validates any tainted input not handled by complete() or flags
// processing.
func (o *createClusterOptions) validate() error {
	return nil
}

// run executes the command.
func (o *createClusterOptions) run() error {
	namespace, err := o.controlPlaneFlags.GetControlPlaneNamespace(context.TODO(), o.client)
	if err != nil {
		return err
	}

	allowedPrefixes := make([]unikornv1.IPv4Prefix, len(o.allowedPrefixes.IPNetworks))

	for i, prefix := range o.allowedPrefixes.IPNetworks {
		allowedPrefixes[i] = unikornv1.IPv4Prefix{
			IPNet: prefix,
		}
	}

	version := unikornv1.SemanticVersion(o.version.Semver)

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: o.name,
			Labels: map[string]string{
				constants.VersionLabel:      constants.Version,
				constants.ProjectLabel:      o.controlPlaneFlags.Project,
				constants.ControlPlaneLabel: o.controlPlaneFlags.ControlPlane,
			},
		},
		Spec: unikornv1.KubernetesClusterSpec{
			ApplicationBundle: &o.applicationBundle,
			Openstack: &unikornv1.KubernetesClusterOpenstackSpec{
				CACert:              &o.caCert,
				CloudConfig:         &o.clouds,
				Cloud:               &o.cloud,
				FailureDomain:       &o.computeAvailabilityZone,
				VolumeFailureDomain: &o.volumeAvailabilityZone,
				SSHKeyName:          &o.sshKeyName,
				ExternalNetworkID:   &o.externalNetworkID,
			},
			Network: &unikornv1.KubernetesClusterNetworkSpec{
				NodeNetwork:    &unikornv1.IPv4Prefix{IPNet: o.nodeNetwork},
				PodNetwork:     &unikornv1.IPv4Prefix{IPNet: o.podNetwork},
				ServiceNetwork: &unikornv1.IPv4Prefix{IPNet: o.serviceNetwork},
				DNSNameservers: unikornv1.IPv4AddressSliceFromIPSlice(o.dnsNameservers),
			},
			API: &unikornv1.KubernetesClusterAPISpec{
				SubjectAlternativeNames: o.SANs,
				AllowedPrefixes:         allowedPrefixes,
			},
			ControlPlane: &unikornv1.KubernetesClusterControlPlaneSpec{
				MachineGeneric: unikornv1.MachineGeneric{
					Version:  &version,
					Image:    &o.image,
					Flavor:   &o.flavor,
					Replicas: &o.replicas,
					DiskSize: o.diskSize.Quantity,
				},
			},
			Features: &unikornv1.KubernetesClusterFeaturesSpec{
				Autoscaling: &o.autoscaling,
				Ingress:     &o.ingress,
			},
		},
	}

	if _, err := o.client.UnikornV1alpha1().KubernetesClusters(namespace).Create(context.TODO(), cluster, metav1.CreateOptions{}); err != nil {
		return err
	}

	fmt.Printf("%s.%s/%s created\n", unikornv1.KubernetesClusterResource, unikornv1.GroupName, o.name)

	return nil
}

var (
	//nolint:gochecknoglobals
	createClusterLong = templates.LongDesc(`
	Create a Kubernetes cluster.

	A cluster is logically an aggregate of a cluster (this command defines the
	cluster control plane), and a set of workload pools (defined with the
	"create workload-pool" command.)

	This command will use standard lookup rules to find a clouds.yaml file on
	your local system.  You need to supply a --cloud parameter to select the
	cloud and user account to provision with.  Only the selected cloud will be
	passed to CAPI for security reasons.  It's also recommended that you use
	the shell completion for --cloud first, as that'll allow further completion
	functions to poll OpenStack to get images, flavors etc.`)

	//nolint:gochecknoglobals
	createClusterExamples = util.TemplatedExample(`
        # Create a Kubernetes cluster
        {{.Application}} create cluster --project foo --control-plane bar --cloud nl1-simon --external-network c9d130bc-301d-45c0-9328-a6964af65579 --flavor c.small --version v1.24.7 --image ubuntu-2004-kube-v1.24.7 --compute-availability-zone nova --volume-availability-zone cinder baz`)
)

// newCreateClusterCommand creates a command that is able to provison a new Kubernetes
// cluster with a Cluster API control plane.
func newCreateClusterCommand(f cmdutil.Factory) *cobra.Command {
	o := &createClusterOptions{}

	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "Create a Kubernetes cluster",
		Long:    createClusterLong,
		Example: createClusterExamples,
		Aliases: aliases.Cluster,
		Run: func(cmd *cobra.Command, args []string) {
			util.AssertNilError(o.complete(f, args))
			util.AssertNilError(o.validate())
			util.AssertNilError(o.run())
		},
	}

	o.addFlags(f, cmd)

	return cmd
}
