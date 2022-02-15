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

# bootstrap
kubectl create -f https://raw.githubusercontent.com/giantswarm/management-clusters-fleet/migrate-to-flux-test/bootstrap/gs-${PROVIDER}/gs-${PROVIDER}.yaml
kubectl create -f backup/$INSTALLATION/github-giantswarm-https-credentials.yaml 
kubectl create -f backup/$INSTALLATION/management-cluster-metadata.yaml 

# cleanup
kubectl label app -n giantswarm --all argocd.argoproj.io/instance-
kubectl label cm -n giantswarm --all argocd.argoproj.io/instance-
kubectl label secret -n giantswarm --all argocd.argoproj.io/instance-

# verify
set +x
for ns in flux-giantswarm flux-system; do
  for d in helm-controller image-automation-controller image-reflector-controller \
  kustomize-controller notification-controller source-controller; do
    kubectl -n $ns wait --for=condition=available --timeout=60s deployment/${d}
  done
done
kubectl -n flux-giantswarm get deploy
kubectl -n flux-system get deploy
echo "*** INFO: all expected Deployments are up"

echo "Checking gitrepos"
for gr in collection giantswarm-config management-clusters-fleet; do
  echo -n "Waiting for GitRepo $gr to be ready"
  while [[ "$(kubectl -n flux-giantswarm get gitrepo $gr --output=jsonpath='{..status.conditions[?(@.type=="Ready")].status}')" != "True" ]]; do
    sleep 1
    echo -n "."
  done
  echo ""
done
echo ""
echo "*** INFO: all expected GitRepos are up"

echo "Checking kustomizations"
for gr in flux customer-flux collection; do
  echo -n "Waiting for Kustomization $gr to be ready"
  while [[ "$(kubectl -n flux-giantswarm get kustomization $gr --output=jsonpath='{..status.conditions[?(@.type=="Ready")].status}')" != "True" ]]; do
    sleep 1
    echo -n "."
  done
  echo ""
done
echo ""
echo "*** INFO: all expected Kustomizations are up"

kubectl -n giantswarm get app --no-headers | grep -v "deployed"
if [[ $? -ne 1 ]]; then
  echo "ERROR: not all Apps in giantswarm namespace are 'deployed'. Check their status."
  exit 1
fi

echo "INFO: All done! All checks OK!"
echo "Remember to restore customer's flux objects now (if there were any) using files from backup/$INSTALLATION/flux-backup"
