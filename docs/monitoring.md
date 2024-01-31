# Monitoring and Logging

All Unikorn controllers support prometheus metrics, and promtail/loki compatible logs out of the box.
Logs in particular are tagged with the instance they relate to and a specific GUID associated with the reconcile request.
As such a nice database is in order to make sense of them (unless you can read JSON...)

## Performance Metrics

When writing provisioners and other things, it's useful to keep a histogram of run-times so that you can spot any anomalies.
This is especially pertinent when OpenStack is in play as it's pretty flaky, and things like load-balancers and ingresses can get stuck very easily.
All metric names should be prefixed with `unikorn_` for easy identification, then some other context to make things totally obvious.

You will most likely want to install Prometheus and Grafana:

```shell
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace
```

This will pretty much work out of the box.
See [`manifests/prometheus.yaml`](https://github.com/unikorn-cloud/unikorn/blob/main/manifests/prometheus.yaml) for an example of use, using the wonderful Prometheus Operator.

### Installing Unikorn Dashboard

There's a rudimentary Grafana dashboard for the Unikorn controller componenets available in tree.
Run [`dashboards/install`](https://github.com/unikorn-cloud/unikorn/blob/main/grafana/install) with a default `kube-prometheus-stack` to automagically have it appear.

## Metrics Server

The prometheus stuff has some good metrics collected by default, and the Unikorn binaries give a decent view into heap allocations.
You could use the `contrainer_cpu_usage_second_total` metric in Grafana for example.
Using the `kubectl top pods` command is a quick and easy way to derive utilisation, hence this section.
If it's not installed by default on your platform:

```shell
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

On KinD, this won't work out of the box as the node certificates don't have an IP SAN.
To fudge this, add the `--kubelet-insecure-tls` flag to the server:

```shell
kubectl edit -n kube-system deployment/metrics-server
```

## OpenTelemetry Tracing

Unikorn makes use of the OTLP protocol to ship information on spans.
This provides a break down of transaction based workloads.

You can install Jaeger's stack with:

```shell
helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
helm upgrade jaeger jaegertracing/jaeger --set collector.service.otlp.http.port=4318
```

## Logs

Coming soon!

For now, use `kubectl logs -f deployment/unikorn-*`.
You can boost the verbosity by adding the follwing to the controll manifests:

```yaml
args:
- -zap-log-level=debug
```

Additionalal options/details you can get from `./bin/amd64-linux-gnu/unikorn-control-plane-manager --help`.
