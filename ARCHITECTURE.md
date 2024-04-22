## Unikorn Resources

Unikorn implements cluster provisioning with 2 resources, nested within a project namespace as described by the [identity service](https://github.com/unikorn-cloud/identity/).
These are:

* Cluster controllers encapsulate cluster life-cycle management e.g. CAPI.
* Kubernetes Clusters that actually create an end-user accessible workload clusters

Because there are no direct references from, for example, a Kubernetes cluster to its owning project and organization, these are required to be added as labels.
This allows the resource to construct a fully-qualified name for provisioning with a CD driver as describe by the [core library](https://github.com/unikorn-cloud/core/).

### Cluster controllers

The server component of this repository will automatically and transparently create a default cluster controller if one does not exist on Kubernetes cluster creation.
You may specify an explicit name during Kubernetes cluster creation and a cluster controller will be created for you.
The intention is to allow clusters to be managed by different controllers, giving rise to horizontal scaling, separate failure domains or different upgrade policies.

Cluster controllers are implemented in a [vCluster](https://www.vcluster.com/) as this allows CAPI to be be installed with minimal fuss, while maintaining separation between different instances.
This includes any prerequisites-requisites required by CAPi.

### Kubernetes Clusters

These are somewhat more involved than cluster managers.

Clusters are composed of charts that define the cluster, any CNI, and cloud controller managers required for cluster bootstrap.

Once those components are up and running we install a generic and minimal set of add-ons such as the cluster autoscaler if required, then any GPU operators, metrics servers, etc.
Additional applications should be provisioned by an application controller.
