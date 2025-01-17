# node-labels-exporter

![Version: 0.1.5](https://img.shields.io/badge/Version-0.1.5-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.3.0](https://img.shields.io/badge/AppVersion-v0.3.0-informational?style=flat-square)

The Node Labels Exporter injects node labels as environment variables into pods.

Applications can then use these environment variables without needing access to the Kubernetes API.

**Homepage:** <https://github.com/sergelogvinov/node-labels-exporter>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| sergelogvinov |  | <https://github.com/sergelogvinov> |

## Source Code

* <https://github.com/sergelogvinov/node-labels-exporter>

## Deploy

```shell
helm upgrade -i -n kube-system node-labels-exporter oci://ghcr.io/sergelogvinov/charts/node-labels-exporter
```

## Usage

Add the following annotations to the pod:

```yaml
annotations:
  # Specify the container name to inject the node labels, separated by commas
  # If not specified, the node labels will be injected into all containers
  node-labels-exporter.sinextra.dev/containers: "alpine,init-container"
  # Specify the node labels to inject the value
  injector.node-labels-exporter.sinextra.dev/zone: "topology.kubernetes.io/zone"
```

After the pod shedduled, it will have the following environment variable:

```shell
ZONE=us-central1-a
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| replicaCount | int | `1` |  |
| image.repository | string | `"ghcr.io/sergelogvinov/node-labels-exporter"` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| fullnameOverride | string | `""` |  |
| args | list | `[]` | Node labels extra arguments. example: --zap-stacktrace-level=info --zap-log-level=debug |
| webhooks | object | `{"failurePolicy":"Ignore","namespaceSelector":{}}` | Admission Control webhooks configuration. ref: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-namespaceselector |
| priorityClassName | string | `"system-cluster-critical"` | Controller pods priorityClassName. |
| serviceAccount | object | `{"annotations":{},"automount":true,"create":true,"name":""}` | Pods Service Account. ref: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/ |
| podAnnotations | object | `{}` | Annotations for controller pod. ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/ |
| podLabels | object | `{}` | Labels to a Pod. ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/ |
| podSecurityContext | object | `{"fsGroup":65532,"fsGroupChangePolicy":"OnRootMismatch","runAsGroup":65532,"runAsNonRoot":true,"runAsUser":65532}` | Controller Security Context. ref: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod |
| securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true,"seccompProfile":{"type":"RuntimeDefault"}}` | Controller Container Security Context. ref: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod |
| updateStrategy | object | `{"rollingUpdate":{"maxSurge":1,"maxUnavailable":0},"type":"RollingUpdate"}` | Controller deployment update strategy type. ref: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#updating-a-deployment |
| service | object | `{"type":"ClusterIP"}` | Service parameters ref: https://kubernetes.io/docs/concepts/services-networking/service/ |
| metrics | object | `{"enabled":false,"port":8080,"type":"annotation"}` | Prometheus metrics |
| metrics.enabled | bool | `false` | Enable Prometheus metrics. |
| metrics.port | int | `8080` | Prometheus metrics port. |
| resources | object | `{"requests":{"cpu":"50m","memory":"64Mi"}}` | Resource requests and limits. ref: https://kubernetes.io/docs/user-guide/compute-resources/ |
| nodeSelector | object | `{}` | Node labels for controller assignment. ref: https://kubernetes.io/docs/user-guide/node-selection/ |
| tolerations | list | `[{"effect":"NoSchedule","key":"node-role.kubernetes.io/control-plane"}]` | Tolerations for controller assignment. ref: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ |
| affinity | object | `{}` | Affinity for controller assignment. ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity |
