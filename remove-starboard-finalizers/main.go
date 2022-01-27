package main

import (
	"context"
	"log"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	schemas = []schema.GroupVersionKind{
		// argo
		{Group: "argoproj.io", Kind: "ApplicationList", Version: "v1alpha1"},
		{Group: "argoproj.io", Kind: "AppProjectList", Version: "v1alpha1"},
		// flux
		{Group: "helm.toolkit.fluxcd.io", Kind: "HelmReleaseList", Version: "v2beta1"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImagePolicyList", Version: "v1alpha2"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImagePolicyList", Version: "v1alpha1"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImagePolicyList", Version: "v1beta1"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImageRepositoryList", Version: "v1beta1"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImageRepositoryList", Version: "v1alpha2"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImageRepositoryList", Version: "v1alpha1"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImageUpdateAutomationList", Version: "v1beta1"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImageUpdateAutomationList", Version: "v1alpha2"},
		{Group: "image.toolkit.fluxcd.io", Kind: "ImageUpdateAutomationList", Version: "v1alpha1"},
		{Group: "kustomize.toolkit.fluxcd.io", Kind: "KustomizationList", Version: "v1beta2"},
		{Group: "kustomize.toolkit.fluxcd.io", Kind: "KustomizationList", Version: "v1beta1"},
		{Group: "notification.toolkit.fluxcd.io", Kind: "AlertList", Version: "v1beta1"},
		{Group: "notification.toolkit.fluxcd.io", Kind: "ProviderList", Version: "v1beta1"},
		{Group: "notification.toolkit.fluxcd.io", Kind: "ReceiverList", Version: "v1beta1"},
		{Group: "source.toolkit.fluxcd.io", Kind: "BucketList", Version: "v1beta1"},
		{Group: "source.toolkit.fluxcd.io", Kind: "GitRepositoryList", Version: "v1beta1"},
		{Group: "source.toolkit.fluxcd.io", Kind: "HelmChartList", Version: "v1beta1"},
		{Group: "source.toolkit.fluxcd.io", Kind: "HelmRepositoryList", Version: "v1beta1"},
		// starboard
		{Group: "aquasecurity.github.io", Kind: "CISKubeBenchReportList", Version: "v1alpha1"},
		{Group: "aquasecurity.github.io", Kind: "ClusterConfigAuditReportList", Version: "v1alpha1"},
		{Group: "aquasecurity.github.io", Kind: "ClusterVulnerabilityReportList", Version: "v1alpha1"},
		{Group: "aquasecurity.github.io", Kind: "ConfigAuditReportList", Version: "v1alpha1"},
		{Group: "aquasecurity.github.io", Kind: "VulnerabilityReportList", Version: "v1alpha1"},
	}

	namespace string = ""
)

func init() {
	flag.StringVar(&namespace, "namespace", "", "namespace to remove starboard finalizers in")
	flag.Parse()
}

func main() {
	if namespace == "" {
		log.Fatal("namespace flag is required")
	}
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Fatal("failed to create client")
	}
	patch := []byte(`[{
"op": "replace",
"path": "/metadata/finalizers",
"value": []
}]`)

	for _, sch := range schemas {
		u := &unstructured.UnstructuredList{}
		u.SetGroupVersionKind(sch)

		err := c.List(context.Background(), u, &client.ListOptions{
			Namespace: namespace,
		})
		if err != nil {
			log.Fatal(err)
		}

		for i, item := range u.Items {
			if item.Object == nil {
				continue
			}
			ptr := &u.Items[i]
			err = c.Patch(
				context.Background(),
				ptr,
				client.RawPatch(types.JSONPatchType, patch),
			)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("  %s/%s", ptr.GetNamespace(), ptr.GetName())
		}
	}
	log.Println("DONE")
}
