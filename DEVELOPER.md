# Unikorn Developer

Developer-centric documentation.

## Installation

_NB_: The Makefile in this repository makes use of conventions specific to GNU/Make.  If you're on a non-GNU system (i.e macOS, FreeBSD) then you need to replace `make` in the examples below with `gmake`.

### Building and Installing the Binaries

Checkout the repository:

```shell
git clone git@github.com:echercloudai/unikorn
cd unikorn
make touch # See the Makefile comments for why
```

Build the binaries and install them:

```shell
make install
```

Please note that the `install` target expects ~/bin to exist and be in your PATH.
You can customize this with `sudo make install -e PREFIX /usr/local/bin` if that is your desire.

#### Setting Up Shell Completion

Obviously this works as `kubectl` does to avoid mistakes, do something like:

```shell
export TEMP=$(mktemp)
unikornctl completion bash > ${TEMP}
source ${TEMP}
```

For the more adventurous, you can add it to `/etc/bash_completion.d/` or whatever you use.

### Creating Docker Images

Images are built via [Docker buildx](https://docs.docker.com/build/buildx/install/), you should install this first in order to be able to reproduce the following steps.

#### Public Cloud

When operating in the Cloud, you'll want to push images to a public registry:

```shell
docker login
make images-push -e DOCKER_ORG=unikorn-cloud VERSION=$(git rev-parse HEAD)
```

Please note, you are using a "non-standard" organization, so will need to alter installation later on.
Additionally, we typically use unique "versions" aka tags for each build, thus avoiding caching.
You may want to just change the image pull policy in the Helm chart (i.e. write said functionality).

Your Helm configuration later should look like `--set repository=null --set organization=unikorn-cloud --set tag=$(git rev-parse HEAD)`.

#### Local Development

If you are doing local development, and using `kind` or similar, you can omit the prior step and use the following:

```shell
make images-kind-load
```

By default, development images will have a tag/version of `0.0.0` so as not to be confused with official releases.

Your Helm configuration later should look like `--set tag=0.0.0`.

### Setting Up Kubernetes

#### Installing

Is all done via Helm.
Remember to override the chart values as described above.
You can install using the local repo:

```shell
helm install unikorn charts/unikorn --namespace unikorn --create-namespace --set repository=null --set tag=0.0.0
```

#### LoadBalancer Service Support

On local development environments, these services aren't supported out of the box.
There's a script provided that will set up Metallb for you if you require e.g. kubectl access to the CAPI control plane:

```shell
go run hack/install_metallb
```
