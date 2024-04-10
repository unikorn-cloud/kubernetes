# Unikorn

![Unikorn Logo](https://raw.githubusercontent.com/unikorn-cloud/assets/main/images/logos/light-on-dark/logo.svg#gh-dark-mode-only)
![Unikorn Logo](https://raw.githubusercontent.com/unikorn-cloud/assets/main/images/logos/dark-on-light/logo.svg#gh-light-mode-only)

## Overview

### Resources

Unikorn abstracts away installation of Cluster API.

There are four resource types:

* Organizations, that are analogous to organizations defined for [Unikorn Identity](https://github.com/unikorn-cloud/identity), but in this component merely provide a namespace for projects.
* Projects, that are a container for higher level abstractions.
* ControlPlanes, that basically are instances of Cluster API that live in Projects.
* Clusters, are Kubernetes clusters, and managed by control planes.

Control planes are actually contained themselves in virtual clusters, this allows horizontal scaling and multi-tenant separation.

Projects allow multiple control planes to be contained within them.
These are useful for providing a boundary for billing etc.

Unsurprisingly, as we are dealing with custom resources, we are managing the lifecycles as Kubernetes controllers ("operator pattern" to those drinking the CoreOS Koolaid).

### Services

Unikorn is split up into domain specific micro-services:

* Project, control plane and cluster controllers.
  These are reactive services that watch for resource changes, then reconcile reality against the requested state.
* Server is a RESTful interface that manages Unikorn resource types.
  It additionally exposes a limited, and opinionated, set of OpenStack interfaces that provide resources that are used to populate required fields in Unikorn resources.
  As it's intended as a public API e.g. for Terraform or a user interface, it integrates authn/authz functionality too.
* UI is a user interface, and provides a seamless and intuitive UX on top of server.
  This adds even more opinionation on top of the REST interface.
  This is hosted in a separate repository.
* Monitor is a daemon that periodically polls Unikorn resource types, and provides functionality that cannot be triggered by reactive controllers.
  Most notably, this includes automatic upgrades.

## Installation

### Installing the Service

Is all done via Helm, which means we can also deploy using ArgoCD.
As this is a private repository, we're keeping the charts private for now also, so you'll need to either checkout the correct branch for a local Helm installation, or imbue Argo with an access token to get access to the repository.

#### Installing ArgoCD

ArgoCD is a **required** to use Unikorn.

Deploy argo using Helm (the release name is _hard coded_, don't change it yet please):

```
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update
helm install argocd argo/argo-cd -n argocd --create-namespace
```

Check the on-screen instructions. To get the admin password you can use:

```
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

#### Installing Prerequisites

The Unikorn server component has a couple prerequisites that are required for correct functionality.
If not installing server you can skip to the next section.

You'll need to install:

* cert-manager (used to generate keying material for JWE/JWS and for ingress TLS)
* nginx-ingress (to perform routing, avoiding CORS, and TLS termination)

<details>
<summary>Helm</summary>

```shell
helm repo add jetstack https://charts.jetstack.io
helm repo add nginx https://helm.nginx.com/stable
helm repo update
helm install cert-manager jetstack/cert-manager --version v1.14.4 -n cert-manager --create-namespace --set installCRDs=true
helm install nginx-ingress nginx/nginx-ingress --version 0.16.1 -n nginx-ingress --create-namespace
```
</details>

<details>
<summary>ArgoCD</summary>

Tip: You can apply these from the web-ui by adding an application then pressing edit as yaml in the top right.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cert-manager
  namespace: argocd
spec:
  project: default
  source:
    chart: cert-manager
    helm:
      parameters:
      - name: installCRDs
        value: "true"
      releaseName: cert-manager
    repoURL: https://charts.jetstack.io
    targetRevision: v1.10.1
  destination:
    name: in-cluster
    namespace: cert-manager
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nginx-ingress
  namespace: argocd
spec:
  project: default
  source:
    chart: nginx-ingress
    helm:
      parameters:
      - name: controller.service.httpPort.enable
        value: "false"
      releaseName: nginx-ingress
    repoURL: https://helm.nginx.com/stable
    targetRevision: 0.16.1
  destination:
    name: in-cluster
    namespace: nginx-ingress
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```
</details>

#### Installing Unikorn

**NOTE**: Unikorn Server is not installed by default, see below for details.

Then install Unikorn:

<details>
<summary>Helm</summary>

```shell
helm install unikorn charts/unikorn --namespace unikorn --create-namespace
```
</details>

<details>
<summary>ArgoCD</summary>

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: unikorn
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://unikorn-cloud.github.io/unikorn
    chart: unikorn
    targetRevision: v0.1.8
  destination:
    namespace: unikorn
    server: https://kubernetes.default.svc
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```
</details>

#### Installing Unikorn Server

To enable it add the parameter `--set server.enabled=true`.
This will install a developer version of the server with a self-signed certificate.

Rudimentary support exists for ACME certificates using the DNS01 protocol.
Only Cloudflare has been implemented and tested.

A typical `values.yaml` that uses ACME could look like:

```yaml
server:
  enabled: true
  ingress:
    host: unikorn.unikorn-cloud.org
  oidc:
    issuer: https://identity.unikorn-cloud.org
```

The host defines the X.509 SAN, and indeed the host the Ingress will respond to.
There is no automated DDNS yet, so you will need to manually add the A record when the ingress comes up.

Unikorn Server uses OIDC (oauth2) to authenticate API requests.
YOu must deploy [Unikorn Identity](https://github.com/unikorn-cloud/identity) first in order to use it.

#### Installing Unikorn UI

To enable Unikorn UI `--set ui.enabled=true`.
This only enables the ingress route for now.
You will also need to install the UI using Helm as described in the [unikorn-ui repository](https://github.com/unikorn-cloud/ui).
It **must** be installed in the same namespace as Unikorn server in order for the service to be seen by the Ingress.

## Monitoring & Logging

* Prometheus monitoring can be enabled with the `--set monitoring.enabled=true` flag.
* OTLP (e.g. Jaeger) tracing can be enabled with the `set server.otlpEndpoint=jaeger-collector.default:4318` flag.

See the [monitoring & logging](docs/monitoring.md) documentation from more information on configuring those services in the first instance..

## Documentation

### Region Providers

These actually provide access to cloud resources and abstract away vendor specifics:

* [OpenStack](pkg/providers/openstack/README.md)

### API (Unikorn Server)

Consult the [server readme](pkg/server/README.md) to get started.

## Development

Consult the [developer documentation](DEVELOPER.md) for local development instructions.

The [architecture documentation](ARCHITECTURE.md) details how it all works, and the design considerations.
