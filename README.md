# cluster-ip
Kubernetes component to determine cluster IP

## Usage

Instalation:
```
kubectl apply -f https://raw.githubusercontent.com/pbochynski/cluster-ip/main/cluster-ip-operator.yaml

Create a resource:

```sh
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
