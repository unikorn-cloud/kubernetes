# Unikorn Server

## Code Generation

Everything is done with an OpenAPI schema.
This allows us to auto-generate the server routing, schema validation middleware, types and clients.
This happens automatically on update via the `Makefile`.
Please ensure updated generated code is commited with your pull request.

## API Definition

Consult the [OpenAPI schema](../../pkg/server/openapi/server.spec.yaml) for full details of what it does.

## Getting Started with Development and Testing.

Once everything is up and running, grab the IP address:

```bash
export INGRESS_ADDR=$(kubectl -n unikorn get ingress/unikorn-server -o 'jsonpath={.status.loadBalancer.ingress[0].ip}')
```
And add it to your resolver:

```bash
echo "${INGRESS_ADDR} unikorn.unikorn-cloud.org" >> /etc/hosts
```
