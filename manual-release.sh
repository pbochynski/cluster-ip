IMAGE_TAG=0.0.11
make docker-build docker-push IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG
make deploy IMG=ghcr.io/pbochynski/cluster-ip:$IMAGE_TAG
kubectl kustomize config/default >cluster-ip-operator.yaml

rm -r ./mod
rm -r ./charts
kyma alpha create module -n kyma-project.io/cluster-ip --version $IMAGE_TAG \
--registry ghcr.io/pbochynski/cluster-ip-module -c pbochynski:$GITHUB_TOKEN \
-o cluster-ip-module-template.yaml 

