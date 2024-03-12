# Unikorn OpenStack Provider

Provides a driver for OpenStack based regions.

## Initial Setup

It is envisoned that an OpenStack cluster may be used for things other than the exclusive use of Unikorn, and as such it tries to respect this as much as possible.
In particular we want to allow different instances of Unikorn to cohabit to support, for example, staging environments.

### OpenStack Platform Configuration

Start by selecting a unique name that will be used for the deployment's name, project, and domain:

```bash
PROJECT=unikorn-staging
PASSWORD=$(apg -m 24)
```

Create the project.
While Unikorn, by necessity, runs as an `admin` account, we use the project to limit scope.
For example while we can see all images and flavors on the system, we can filter out those not shared explicitly with the project.
Likewise, we can have project specific images that aren't shared with other users on the system, this allows then to be discovered by the provider.
By marking the images as `community` we can then use the images for deployment in other per-Kubernetes cluster projects.

```bash
PROJECT_ID=$(openstack create project ${PROJECT} -f json | jq -r .id}
```

Crete the user.

```bash
USER_ID=$(openstack user create --project ${PROJECT} --password=${PASSWORD} ${PROJECT} -f json | jq -r .id)
```

Grant any roles to the user.
When a Kubernetes cluster is provisioned, it will be done using application credentials, so ensure any required application credentials as configured for the region are explicitly associated with the user here.

```bash
openstack role add --user ${USER_ID} --project ${PROJECT_ID} admin
openstack role add --user ${USER_ID} --project ${PROJECT_ID} member
openstack role add --user ${USER_ID} --project ${PROJECT_ID} load-balancer_member
```

Create the domain.
The use of project domains for projects deployed to provision Kubernetes cluster acheives a few aims.
First namespace isolation.
Second is a security consideration.
It is dangerous, anecdotally, to have a privileged process that has the power of deletion.
By limiting the scope of list operations to that of the project domain we limit our impact on other tenants on the system.
A domain may also aid in simplifying operations like auditing and capacity planning.

```bash
DOMAIN_ID=$(openstack domain create ${PROJECT} -f json | jq -r .id}
```

### Unikorn Configuration

When we create a `Region` of type `openstack`, it will require a secret that contains credentials.
This can be configured as follows.

```bash
kubectl create secret -n unikorn uk-north-1-credentials \
	--from-literal=domain-id=${DOMAIN_ID} \
	--from-literal=user-id=${USER_ID} \
	--from-literal=password=${PASSWORD}
```

Finally we can create the region itself.
For additional configuration options for individual OpenStack services, consult `kubectl explain regions.unikorn-cloud.org` for documentation.

```yaml
apiVersion: unikorn-cloud.org/v1alpha1
kind: Region
metadata:
  name: uk-north-1
spec:
  provider: openstack
  openstack:
    endpoint: https://openstack.uk-north-1.unikorn-cloud.org:5000
    serviceAccountSecret:
      namespace: unikorn
      name: uk-north-1-credentials
```
