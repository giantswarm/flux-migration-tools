# flux-migration-tools

All tools use current kubecontext to perform operations on cluster.

## backup

Stores all ArgoCD and FluxCD resources to local directories.

Example run:
```
backup/ go run .
2022/01/12 12:04:33 Backing up Argo resources...
2022/01/12 12:04:33 Backed up 54 argoproj.io/v1alpha1 Application
2022/01/12 12:04:33 Backed up 7 argoproj.io/v1alpha1 AppProject
2022/01/12 12:04:33 Backing up Flux resources...
2022/01/12 12:04:33 Backed up 2 helm.toolkit.fluxcd.io/v2beta1 HelmRelease
2022/01/12 12:04:33 Backed up 1 image.toolkit.fluxcd.io/v1alpha2 ImagePolicy
2022/01/12 12:04:33 Backed up 1 image.toolkit.fluxcd.io/v1beta1 ImageRepository
2022/01/12 12:04:34 Backed up 1 image.toolkit.fluxcd.io/v1beta1 ImageUpdateAutomation
2022/01/12 12:04:34 Backed up 6 kustomize.toolkit.fluxcd.io/v1beta2 Kustomization
2022/01/12 12:04:34 Backed up 3 source.toolkit.fluxcd.io/v1beta1 GitRepository
2022/01/12 12:04:34 Backed up 2 source.toolkit.fluxcd.io/v1beta1 HelmChart
2022/01/12 12:04:34 Backed up 2 source.toolkit.fluxcd.io/v1beta1 HelmRepository

backup/ ls argo-backup/
argoproj.io.v1alpha1.applicationlist.yaml  argoproj.io.v1alpha1.appprojectlist.yaml

backup/ ls flux-backup/
helm.toolkit.fluxcd.io.v2beta1.helmreleaselist.yaml
image.toolkit.fluxcd.io.v1alpha2.imagepolicylist.yaml
image.toolkit.fluxcd.io.v1beta1.imagerepositorylist.yaml
image.toolkit.fluxcd.io.v1beta1.imageupdateautomationlist.yaml
kustomize.toolkit.fluxcd.io.v1beta2.kustomizationlist.yaml
source.toolkit.fluxcd.io.v1beta1.gitrepositorylist.yaml
source.toolkit.fluxcd.io.v1beta1.helmchartlist.yaml
source.toolkit.fluxcd.io.v1beta1.helmrepositorylist.yaml
```

## convert-collection

Converts Argo Applications found in `*-app-collection` to valid Flux
Kustomizations with `konfigure` plugin.

Usage:
```
convert-collection/ go run . --help
Usage of flux-collection-migration:
      --collection-dir string   path to a directory containing the collection
      --dir string              target directory, inside collections directory, where all new resources will be saved (default "manifests")
```

## pause-argo

Changes `.spec.syncPolicy.automated.selfHeal` and `prune` to `false` for all
Argo Application resources.

Example run:
```
pause-argo/ go run .
2022/01/12 12:09:19 Pausing 54 Argo Applications...
(...)
2022/01/12 12:09:20   argocd/cert-exporter
2022/01/12 12:09:24   argocd/vertical-pod-autoscaler-app
2022/01/12 12:09:24 DONE
```

## remove-finalizers

Strips finalizers from Argo, Flux, and Starboard CRs. Can be used to get rid of
CRs stuck in deletion after operators have already been scaled down.
