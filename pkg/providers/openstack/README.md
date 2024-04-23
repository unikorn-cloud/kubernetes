# Unikorn OpenStack Provider

Provides a driver for OpenStack based regions.

## Initial Setup

It is envisoned that an OpenStack cluster may be used for things other than the exclusive use of Unikorn, and as such it tries to respect this as much as possible.
In particular we want to allow different instances of Unikorn to cohabit to support, for example, staging environments.

You will need to install the [domain manager](https://docs.scs.community/standards/scs-0302-v1-domain-manager-role/) policy defined by SCS.
You will also need to edit this to allow the `_member_` role to be granted.

### OpenStack Platform Configuration

Start by selecting a unique name that will be used for the deployment's name, project, and domain:

```bash
export USER=unikorn-staging
export DOMAIN=unikorn-staging
export PASSWORD=$(apg -n 1 -m 24)
```

Create the domain.
The use of project domains for projects deployed to provision Kubernetes cluster acheives a few aims.
First namespace isolation.
Second is a security consideration.
It is dangerous, anecdotally, to have a privileged process that has the power of deletion.
By limiting the scope of list operations to that of the project domain we limit our impact on other tenants on the system.
A domain may also aid in simplifying operations like auditing and capacity planning.

```bash
DOMAIN_ID=$(openstack domain create ${DOMAIN} -f json | jq -r .id)
```

Crete the user.

```bash
USER_ID=$(openstack user create --domain ${DOMAIN_ID} --password ${PASSWORD} ${USER} -f json | jq -r .id)
```

Grant any roles to the user.
When a Kubernetes cluster is provisioned, it will be done using application credentials, so ensure any required application credentials as configured for the region are explicitly associated with the user here.

```bash
for role in _member_ member load-balancer_member manager; do
	openstack role add --user ${USER_ID} --domain ${DOMAIN_ID} ${role}
done
```

### Unikorn Configuration

When we create a `Region` of type `openstack`, it will require a secret that contains credentials.
This can be configured as follows.

```bash
kubectl create secret generic -n unikorn uk-north-1-credentials \
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

Cleanup actions.

```bash
unset DOMAIN_ID
unset USER_ID
unset PASSWORD
unset DOMAIN
unset USER
```
