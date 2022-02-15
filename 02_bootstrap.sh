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
kubectl -n flux-giantswarm get deploy
kubectl -n flux-system get deploy
kubectl -n flux-giantswarm get gitrepo,kustomization
kubectl -n giantswarm get app | grep -v "deployed"
