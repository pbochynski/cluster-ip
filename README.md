# cluster-ip
Kubernetes component to determine cluster IP which in most cases is a list of external IPs of cluster nodes. 

| WARNING: the `cluster-ip` operator is experimental and you can use it on your own risk. It is also not part of [kyma-project](https://kyma-project.io) yet (maybe in the future) |
| --- |

## Instalation

Install cluster-ip operator:
```
kubectl apply -f https://github.com/pbochynski/cluster-ip/releases/latest/download/cluster-ip-operator.yaml
```{{exec}}
It installs the [latest version](https://github.com/pbochynski/cluster-ip/releases/latest), but you can pick any [other version](https://github.com/pbochynski/cluster-ip/releases). 

## Usage

### Get IPs of all nodes
In this case `cluster-ip` operator will deploy worker in each distinct node using nodeSelector with `kubernetes.io/hostname` label. In the result you will get external IP addresses of all nodes.

Create a resource:

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: ClusterIP
metadata:
  name: clusterip-sample
spec:
  nodeSpreadLabel: kubernetes.io/hostname
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
  - ip: 74.234.131.27
    lastUpdateTime: "2023-02-16T10:02:08Z"
    nodeLabel: shoot--xxxx-l7rs5
  - ip: 74.234.189.156
    lastUpdateTime: "2023-02-16T10:02:08Z"
    nodeLabel: shoot--xxxx-9676m
  - ip: 108.143.196.141
    lastUpdateTime: "2023-02-16T10:02:08Z"
    nodeLabel: shoot--xxxx-7dtg4
  state: Ready
```

You can extract all the IPs in all availability zones using this command
```sh
kubectl get clusterips/clusterip-sample -ojson | jq -r '.status.nodeIPs[].ip'
```
with such output:
```
4.234.131.27
74.234.189.156
108.143.196.141
```
### Multizone scenario with NAT Gateway per availability zone

In this case, we assume that all nodes in the availability zone share the NAT gateway and have the same external IP address. Therefore it is enough to deploy one worker per availability zone using the nodeSelector with the standard `topology.kubernetes.io/zone` label.

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: ClusterIP
metadata:
  name: clusterip-sample
spec:
  nodeSpreadLabel: topology.kubernetes.io/zone
EOF
```

Here you can see how what will happen if you have 3 node cluster in 2 availability zones:

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

## Clean up

You can remove the operator and all the resources with:
```
kubectl delete -f https://raw.githubusercontent.com/pbochynski/cluster-ip/main/cluster-ip-operator.yaml
```
