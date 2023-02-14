# cluster-ip
Kubernetes component to determine cluster IP

## Usage

Create a resource:

```sh
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: ClusterIP
metadata:
  name: clusterip-sample
spec:
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

...

status:
  state: Ready
  zones:
  - ip: 74.234.189.156
    lastUpdateTime: "2023-02-14T08:57:09Z"
    zone: westeurope-3
  - ip: 108.143.196.141
    lastUpdateTime: "2023-02-14T08:57:09Z"
    zone: westeurope-1
  - ip: 74.234.131.27
    lastUpdateTime: "2023-02-14T08:57:09Z"
    zone: westeurope-2
```

You can extract all the IPs in all availability zones using this command
```sh
 kubectl get clusterips/clusterip-sample -ojson | jq -r '.status.zones[].ip'
```
with such output:
```
74.234.189.156
108.143.196.141
74.234.131.27
```
