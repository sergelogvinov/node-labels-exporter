# Install controller

## Install by using kubectl

Install latest release version

```shell
kubectl apply -f https://raw.githubusercontent.com/sergelogvinov/node-labels-exporter/refs/heads/main/docs/deploy/node-labels-exporter-release.yml
```

Or install latest stable version (edge)

```shell
kubectl apply -f https://raw.githubusercontent.com/sergelogvinov/node-labels-exporter/refs/heads/main/docs/deploy/node-labels-exporter.yml
```

### Install by using Helm

Create the helm values file, for more information see [values.yaml](../charts/node-labels-exporter/values.yaml)

```shell
helm upgrade -i -n kube-system node-labels-exporter oci://ghcr.io/sergelogvinov/charts/node-labels-exporter
```

### Install the plugin by using Talos machine config

If you're running [Talos](https://www.talos.dev/) you can install Hybrid CSI plugin using the machine config

```yaml
cluster:
  externalCloudProvider:
    enabled: true
    manifests:
      - https://raw.githubusercontent.com/sergelogvinov/node-labels-exporter/refs/heads/main/docs/deploy/node-labels-exporter.yml
```
