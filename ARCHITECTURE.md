## Unikorn Resources

Unikorn implements cluster provisioning with 3 resources that are nested in the given order:

* Projects encapsulate the relationship between Unikorn and a tenancy on a multi-tenant cloud
* Control Planes encapsulate cluster management e.g. CAPI.
  These are kept deliberately separate from clusters to provide enviroments e.g. production, staging, that can be independently upgraded canary style
* Kubernetes Clusters that actually create an end-user accessible workload clusters

Conceptually these resources form a forest as shown below:

![Unikorn CRD Hierarchy](docs/arch_crd_hierarchy.png)

Projects are globally scoped, so one instance of Unikorn is associated with one cloud provider.

Both projects and control planes dynamically allocate Kubernetes namespaces, allowing reuse of names for control planes across projects, and clusters across control planes.
In layman's terms project A can have a control plane called X, and project B can have a control planes called X too.

This is shown below:

![Unikorn CR Namespaces](docs/arch_namespaces.png)

Because there are no direct references from, for example, a Kubernetes cluster to its owning project or control plane, these are required to be added as labels.
The labels can be used to construct globally unique fully-qualified names, that include a combination of project, control plane and cluster.

Likewise, the namespaces use generated names, so also need labels to uniquely identify them when performing a lookup operation.
Additionally, namespaces have a kind label, otherwise control plane namespaces will alias with their owning project as the latter's labels are a subset of the former.
