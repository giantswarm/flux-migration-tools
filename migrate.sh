#!/bin/bash

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 [INSTALLATION_NAME] [PROVIDER_NAME] [USER_EMAIL]"
  echo " Example: $0 talos azure lukasz@giantswarm.io"
  exit 1
fi

INSTALLATION=$1
PROVIDER=$2
EMAIL=$3

set -x
set -e

exit 0

# login
opsctl kgs login -i $INSTALLATION

# alerts
opsctl create routingrule -u $EMAIL -c $INSTALLATION -r '.*' -n "$INSTALLATION-argo-to-flux-$USER" --ttl 2h

# vault
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
mkdir $INSTALLATION; mv argo-backup flux-backup $INSTALLATION
kubectl get cm -n argocd management-cluster-metadata -o yaml | kubectl neat > management-cluster-metadata.yaml
kubectl get secret -n argocd github-giantswarm-https-credentials -o yaml | kubectl neat > github-giantswarm-https-credentials.yaml
sed -i 's/namespace: argocd/namespace: flux-giantswarm/' *.yaml
sed -i "2 a \ \ CLUSTER_DOMAIN: ${cluster_domain}" management-cluster-metadata.yaml
mv *.yaml backup/$INSTALLATION
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
kubectl delete -f https://raw.githubusercontent.com/giantswarm/management-clusters-fleet/main/bootstrap/${PROVIDER}.yaml

# remove finalizers
cd remove-finalizers/
go run .
cd ..
cd remove-starboard-finalizers
go run . --namespace flux-system
go run . --namespace argocd
cd ..

# remove flux CRDs
kubectl delete crd $(kubectl get crd | grep 'toolkit.fluxcd.io' | cut -f1 -d" ")

# delete argo
kubectl -n argocd delete application --all
kubectl delete ns argocd

# bootstrap
kubectl create -f https://raw.githubusercontent.com/giantswarm/management-clusters-fleet/migrate-to-flux-test/bootstrap/gs-${PROVIDER}/gs-${PROVIDER}.yaml
kubectl create -f backup/$INSTALLATION/github-giantswarm-https-credentials.yaml 
kubectl create -f backup/$INSTALLATION/management-cluster-metadata.yaml 

# cleanup
kubectl label app -n giantswarm --all argocd.argoproj.io/instance-
kubectl label cm -n giantswarm --all argocd.argoproj.io/instance-
kubectl label secret -n giantswarm --all argocd.argoproj.io/instance-

# verify
kubectl -n flux-giantswarm get deploy
kubectl -n flux-system get deploy
kubectl -n flux-giantswarm get gitrepo,kustomization
kubectl -n giantswarm get app | grep -v "deployed"
