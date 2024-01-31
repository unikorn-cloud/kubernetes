# Unikorn

![Unikorn Logo](https://raw.githubusercontent.com/unikorn-cloud/assets/main/images/logos/light-on-dark/logo.svg#gh-dark-mode-only)
![Unikorn Logo](https://raw.githubusercontent.com/unikorn-cloud/assets/main/images/logos/dark-on-light/logo.svg#gh-light-mode-only)

## Overview

### Resources

Unikorn abstracts away installation of Cluster API.

There are three resource types:

* Projects, that are a container for higher level abstractions.
* ControlPlanes, that basically are instances of Cluster API that live in Projects.
* Clusters, Kubernetes clusters.

Control planes are actually contained themselves in virtual clusters, as CAPI is pretty terrible at cleaning things up on OpenStack errors, so we make these cattle.
One Kubernetes cluster to one instance of Cluster API.
If something goes wrong, just delete the virtual cluster and restart.
In future, when things get more stable, we can support many-to-one to save on resource costs, and even do away with virtual clusters entirely.

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

### Installing the Management Binary

**NOTE**: You are better off installing Unikorn Server/UI as a shell, see below for details.

Download the official binary (update the version as appropriate):

```shell
wget -O ~/bin/unikornctl https://github.com/unikorn-cloud/unikorn/releases/download/v0.4.1/unikornctl-linux-amd64
```

### Set up shell completion

`unikornctl` has a large amount of convenience and contextual awareness built into its various subcommands.  It's strongly recommended to set up shell completion to make your life a lot easier.

<details>
<summary>BASH</summary>

```shell
export TEMP=$(mktemp)
unikornctl completion bash > ${TEMP}
source ${TEMP}
```

For the more adventurous, you can add it to `/etc/bash_completion.d/` or whatever you use.
</details>

<details>
<summary>ZSH</summary>

With zsh, the [recommendation](https://jzelinskie.com/posts/dont-recommend-sourcing-shell-completion/) is to do the following:

```shell
autoload -U +X compinit && compinit
unikornctl completion zsh > $fpath/_unikornctl
```

If you have a set of existing paths in `$fpath`, create the `_unikornctl` in your own custom completion function directory.  For example, if you had custom functions in `~/.zshfunc` then you would add the following to your `~/.zshenv`:

```
fpath=( ~/.zshfunc "${fpath[@]}" )
```

And then redirect the output of `unikornctl completion zsh` to `~/.zshfunc/_unikornctl`.
</details>

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
helm install cert-manager jetstack/cert-manager -v v1.10.1 -n cert-manager --create-namespace
helm install nginx-ingress nginx/nginx-ingress -v 0.16.1 -n nginx-ingress --create-namespace
```
</details>

<details>
<summary>ArgoCD</summary>

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
    targetRevision: v0.4.1
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
  host: kubernetes.my-domain.com
  acme:
    email: spam@my-domain.com
    server: https://acme-v02.api.letsencrypt.org/directory
    cloudflare:
      email: cloudflare-user@my-domain.com
      apiToken: bW92ZV9hbG9uZ19jbGV2ZXJfZGljaw==
```

The host defines the X.509 SAN, and indeed the host the Ingress will respond to.
There is no automated DDNS yet, so you will need to manually add the A record when the ingress comes up.

The server shown is the production server, and will be trusted by browsers.
You can use the `acme-staging-v02` host if just playing around as it's not as badly rate limited.

Finally the API token you will need to configure to allow editing of records in your domain.
It's base64 encoded e.g. `echo -n MY_TOKEN | base64 -w0`.

#### Installing Unikorn UI

To enable Unikorn UI `--set ui.enabled=true`.
This only enables the ingress route for now.
You will also need to install the UI using Helm as described in the [unikorn-ui repository](https://github.com/unikorn-cloud/unikorn-ui).
It **must** be installed in the same namespace as Unikorn server in order for the service to be seen by the Ingress.

## Monitoring & Logging

* Prometheus monitoring can be enabled with the `--set monitoring.enabled=true` flag.
* OTLP (e.g. Jaeger) tracing can be enabled with the `set server.otlpEndpoint=jaeger-collector.default:4318` flag.

See the [monitoring & logging](docs/monitoring.md) documentation from more information on configuring those services in the first instance..

## Documentation

### CLI

All the best tools document themselves, try:

```shell
unikornctl --help
unikornctl create --help
```

### API (Unikorn Server)

Consult the [server readme](pkg/server/README.md) to get started.

## Development

Consult the [developer documentation](DEVELOPER.md) for local development instructions.

The [architecture documentation](ARCHITECTURE.md) details how it all works, and the design considerations.
