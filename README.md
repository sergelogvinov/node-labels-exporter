# Node Labels Exporter

The Node Labels Exporter is designed for hybrid cloud environments where nodes are in different cloud providers or on-premises, and the cluster spans multiple zones and regions. Applications need to understand the cluster topology to make the right decisions. To discover the cluster topology, the applications need cluster role permissions. However, granting cluster role permissions to applications is not always safe.

The Node Labels Exporter injects node labels as environment variables into the pods. Applications can then use these environment variables to retrieve the cluster topology.

Using pod annotations, you can specify which node labels to export to the pods.

```yaml
annotations:
  injector.node-labels-exporter.sinextra.dev/zone: "topology.kubernetes.io/zone"
  injector.node-labels-exporter.sinextra.dev/node-pool: "node.kubernetes.io/instance-type"
```

In the POD, the environment variables will be:

* `ZONE` - is environment variable with the value of the node label `topology.kubernetes.io/zone`
* `NODE_POOL` - is environment variable with the value of the node label `node.kubernetes.io/instance-type`

All environment variables are transformed to uppercase and the `-` is replaced by `_`.

## Installation

Install the Node Labels Exporter in your cluster. The Kubernetes API will call the Node Labels Exporter service to set the environment variables in the pods. If possible, install the Node Labels Exporter in the control plane.

### Helm

```shell
helm upgrade -i -n kube-system node-labels-exporter oci://ghcr.io/sergelogvinov/charts/node-labels-exporter
```

### Kubectl

```shell
kubectl apply -f https://raw.githubusercontent.com/sergelogvinov/node-labels-exporter/refs/heads/main/docs/deploy/node-labels-exporter-release.yml
```

## Deployment examples

Deploy a test statefulSet, it already has annotations for environment.

```yaml
  template:
    metadata:
      annotations:
        injector.node-labels-exporter.sinextra.dev/zone: "topology.kubernetes.io/zone"
        injector.node-labels-exporter.sinextra.dev/node-pool: "node.kubernetes.io/instance-type"
```

```shell
kubectl apply -f https://raw.githubusercontent.com/sergelogvinov/node-labels-exporter/refs/heads/main/docs/deploy/test-statefulset.yaml
```

Check the pods, they should be running on different nodes.

```shell
kubectl -n default get po -owide
```

```shell
NAME     READY   STATUS    RESTARTS   AGE   IP             NODE       NOMINATED NODE   READINESS GATES
test-0   1/1     Running   0          5s    10.32.18.181   kube-08a   <none>           <none>
test-1   1/1     Running   0          11s   10.32.3.167    kube-01b   <none>           <none>
```

Check the environment in the pod:

```shell
kubectl -n default exec -ti test-0 -- env
```

Output:

```shell
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
HOSTNAME=test-0
NODE_POOL=12VCPU-56GB
ZONE=pve-2
```

The pod has the environment variables `NODE_POOL` and `ZONE` with the values of the node labels.

## Resources

* [Kubernetes Admission Controllers](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/)
* [Matching Requests NamespaceSelector](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-namespaceselector)

## Contributing

Contributions are welcomed and appreciated!
See [Contributing](CONTRIBUTING.md) for our guidelines.

## License

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
