#!/bin/bash

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 [INSTALLATION_NAME] [PROVIDER_NAME] [USER_EMAIL]"
  echo " Example: $0 talos azure lukasz@giantswarm.io"
  exit 1
fi

INSTALLATION=$1
PROVIDER=$2
EMAIL=$3

echo "I'm going to do migration magic for installation $INSTALLATION on provider $PROVIDER using email $EMAIL for alerts."
echo "Type 'yolo' below to continue."
read yolo

if [[ "$yolo" != "yolo" ]]; then
  echo "Exiting, as you're not really YOLO."
  exit 2
fi

set -x
set -e

# login
opsctl kgs login -i $INSTALLATION

# alerts
opsctl create routingrule -u $EMAIL -c $INSTALLATION -r '.*' -n "$INSTALLATION-argo-to-flux-$USER" --ttl 2h

# vault
unset VAULT_ADDR
unset VAULT_TOKEN
unset VAULT_CAPATH
$(opsctl create vaultconfig -i $INSTALLATION | tail -n 4)
vault read auth/kubernetes/role/konfigure
vault write auth/kubernetes/role/konfigure \
bound_service_account_names="*" \
bound_service_account_namespaces=flux-giantswarm \
 policies=konfigure \
 ttl=4320h

# backup
cd backup/
go run .
mkdir $INSTALLATION; mv argo-backup flux-backup flux-config-backup $INSTALLATION
kubectl get cm -n argocd management-cluster-metadata -o yaml | kubectl neat > management-cluster-metadata.yaml
kubectl get secret -n argocd github-giantswarm-https-credentials -o yaml | kubectl neat > github-giantswarm-https-credentials.yaml
sed -i 's/namespace: argocd/namespace: flux-giantswarm/' *.yaml
cluster_domain=$(kubectl -n kube-system exec -it $(ks get po -l app=nginx-ingress-controller | head -n 2 | tail -n 1 | cut -f1 -d" ") -- cat /etc/resolv.conf | egrep "^search" | cut -f4 -d" ")
echo "Detected cluster domain: $cluster_domain"
sed -i "2 a \ \ CLUSTER_DOMAIN: ${cluster_domain}" management-cluster-metadata.yaml
mv *.yaml $INSTALLATION
cd ..

# pause argo
cd pause-argo/
go run .
go run .
cd ..

# pause flux
for ns in $(flux get all -A | cut -f1 | egrep -v "NAMESPACE|^$" | sort | uniq); do
   for cr in alert helmrelease kustomization receiver; do
      flux suspend -n $ns --all $cr;
   done;
   for cr in repository update; do
      flux suspend -n $ns --all image $cr;
   done;
   for cr in bucket chart git helm; do
      flux suspend -n $ns --all source $cr;
   done;
done;

# scale down
kubectl -n argocd scale statefulset argocd-application-controller --replicas 0
for d in argocd-redis argocd-repo-server argocd-server; do
   kubectl -n argocd scale deployment $d --replicas 0;
done;
for d in helm-controller image-automation-controller image-reflector-controller \
kustomize-controller notification-controller source-controller; do
   kubectl -n flux-system scale deployment $d --replicas 0;
done;

# uninstall argo
# returns error if anything is not found, so
set +e
kubectl delete -f https://raw.githubusercontent.com/giantswarm/management-clusters-fleet/main/bootstrap/${PROVIDER}.yaml
set -e

# remove finalizers
cd remove-finalizers/
go run .
cd ..
cd remove-starboard-finalizers
go run . --namespace flux-system
go run . --namespace argocd
cd ..

# remove flux CRDs
set +e
kubectl delete clusterrolebindings crd-controller
set -e
kubectl delete crd $(kubectl get crd | grep 'toolkit.fluxcd.io' | cut -f1 -d" ")

# delete argo
kubectl -n argocd delete application --all
kubectl delete ns argocd

# verify
set +x
kubectl get ns argocd
if [[ $? -ne 1 ]]; then
  echo "ERROR: argocd namespace still exists"
  exit 1
fi
kubectl get ns flux-system
if [[ $? -ne 1 ]]; then
  echo "ERROR: flux-system namespace still exists"
  exit 1
fi
kubectl get crd | egrep "fluxcd.io|argoproj.io"
if [[ $? -ne 1 ]]; then
  echo "ERROR: some argo or flux CRDs still exist"
  exit 1
fi

echo "***"
echo "All done! Cluster seems to be ready, run manual checks then run 02_bootstrap.sh"
