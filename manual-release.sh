IMAGE_TAG=0.0.14
make docker-build docker-push IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG
make deploy IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG
kubectl kustomize config/default >cluster-ip-operator.yaml

rm -r ./mod
rm -r ./charts
kyma alpha create module -n kyma-project.io/cluster-ip --version $IMAGE_TAG \
--registry ghcr.io/pbochynski/cluster-ip-module -c pbochynski:$GITHUB_TOKEN \
-o cluster-ip-module-template.yaml 

curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.24.10+k3s1" K3S_KUBECONFIG_MODE=600 INSTALL_K3S_EXEC="server --disable traefik" sh -

git clone https://github.com/pbochynski/cluster-ip.git
cd cluster-ip
export IMAGE_TAG=$(git rev-parse --short HEAD) 
make install
make docker-build IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG
make deploy IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG

curl -s https://raw.githubusercontent.com/rancher/k3d/main/install.sh | bash
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
k3d cluster create envtest -a 1 -i "rancher/k3s:v1.24.10-k3s1" \
--k3s-arg "--disable=traefik@server:0" \
--k3s-node-label "topology.kubernetes.io/zone=zone-a@server:0" \
--k3s-node-label "topology.kubernetes.io/zone=zone-b@agent:0" \

k3d image import ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG -c envtest


curl -sfL https://get.k3s.io | sh -s - server --docker  --token=SECRET --https-listen-port=5555
K3S_TOKEN=SECRET k3s agent --server https://127.0.0.1:5555 