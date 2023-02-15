# cluster-ip
Kubernetes component to determine cluster IP

## Usage

Instalation:
```
kubectl apply -f https://raw.githubusercontent.com/pbochynski/cluster-ip/main/cluster-ip-operator.yaml
```

Create a resource:

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: ClusterIP
metadata:
  name: clusterip-sample
spec:
  nodeSpreadLabel: kubernetes.io/hostname  # topology.kubernetes.io/zone is default
EOF
```

Wait until cluster IP resource is ready:
```sh
kubectl wait --for=jsonpath='{.status.state}'=Ready  clusterips/clusterip-sample
```

Check the status:
```sh
kubectl get clusterips/clusterip-sample -oyaml
```

The result is similar to:
```yaml
apiVersion: operator.kyma-project.io/v1alpha1
kind: ClusterIP
metadata:
  name: clusterip-sample
  namespace: default
spec:
  nodeSpreadLabel: kubernetes.io/hostname
status:
  nodeIPs:
  - ip: 141.95.98.214
    lastUpdateTime: "2023-02-15T19:36:16Z"
```

You can extract all the IPs in all availability zones using this command
```sh
kubectl get clusterips/clusterip-sample -ojson | jq -r '.status.nodeIPs[].ip'
```
with such output:
```
141.95.98.214
```

## How does it work?


```
                       ┌───────────────────────┐
                       │       zone a          │
                       │ ┌─────────────────┐   │
                       │ │ hostname: node1 │   │
                       │ │ zone: a         │   │
                       │ ├─────────────────┤   │
                       │ │   cluster-ip    │   │   manage      ┌────────────────────────────────┐
                       │ │    operator     ├───┼──────────────►│kind: ClusterIP                 │
                       │ ├─────────────────┤   │               │metadata:                       │
                       │ │   cluster-ip    │   │               │  name: clusterip-sample        │
                    ┌──┼─┤     worker      ├───┼──────┐        │spec:                           │
                    │  │ └─────────────────┘   │      │        │  nodeSpreadLabel:              │
┌───────────────┐   │  │                       │      │        │    topology.kubernetes.io/zone │
│ External host │◄──┤  └───────────────────────┘    update IP  │status:                         │
│ (ifconfig.me) │   │                                 │        │  state: Ready                  │
└───────────────┘   │                                 │        │  nodeIPs:                      │
                    │  ┌───────────────────────┐      │        │  - ip: 74.234.131.27           │
                    │  │       zone b          │      └────────┼──► lastUpdateTime: "23:08:12Z" │
                    │  │ ┌─────────────────┐   │               │    nodeLabel: zone-a           │
                    │  │ │ hostname: node2 │   │               │  - ip: 74.234.189.156          │
                    │  │ │ zone: b         │   │      ┌────────┼──► lastUpdateTime: "23:08:12Z" │
                    │  │ ├─────────────────┤   │      │        │    nodeLabel: zone-b           │
                    │  │ │   cluster-ip    │   │      │        └────────────────────────────────┘
                    └──┼─┤     worker      ├───┼──────┘
                       │ └─────────────────┘   │   update IP
                       │ ┌─────────────────┐   │
                       │ │ hostname: node3 │   │
                       │ │ zone: b         │   │
                       │ └─────────────────┘   │
                       └───────────────────────┘
```