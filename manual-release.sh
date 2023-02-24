IMAGE_TAG=0.0.13
make docker-build docker-push IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG
make deploy IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG
kubectl kustomize config/default >cluster-ip-operator.yaml

rm -r ./mod
rm -r ./charts
kyma alpha create module -n kyma-project.io/cluster-ip --version $IMAGE_TAG \
--registry ghcr.io/pbochynski/cluster-ip-module -c pbochynski:$GITHUB_TOKEN \
-o cluster-ip-module-template.yaml 

curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.24.10+k3s1" K3S_KUBECONFIG_MODE=600 INSTALL_K3S_EXEC="server --disable traefik" sh -

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
export IMAGE_TAG=$(git rev-parse --short HEAD) 
make IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG