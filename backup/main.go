package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

const (
	argoDir = "argo-backup"
	fluxDir = "flux-backup"
)

var (
	argoSchemas = []schema.GroupVersionKind{
		{Group: "argoproj.io", Kind: "ApplicationList", Version: "v1alpha1"},
		{Group: "argoproj.io", Kind: "AppProjectList", Version: "v1alpha1"},
	}
	// fluxSchemas are ORDERED from the newest API version to the oldest. This
	// is very important, as we rely on getting CR once, with the latest API
	// version, regardless of how many versions are served.
	fluxSchemas = []schema.GroupVersionKind{
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
	}
)

func main() {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Fatal("failed to create client")
	}

	if err := os.MkdirAll(fluxDir, 0755); err != nil {
		log.Fatalf("failed to create %s directory", fluxDir)
	}
	if err := os.MkdirAll(argoDir, 0755); err != nil {
		log.Fatalf("failed to create %s directory", argoDir)
	}

	log.Println("Backing up Argo resources...")
	if err := backup(c, argoDir, argoSchemas); err != nil {
		log.Fatal(err)
	}

	log.Println("Backing up Flux resources...")
	if err := backup(c, fluxDir, fluxSchemas); err != nil {
		log.Fatal(err)
	}

}

func backup(c client.Client, dir string, schemas []schema.GroupVersionKind) error {
	seenUIDs := map[string]bool{}

	for _, sch := range schemas {
		u := &unstructured.UnstructuredList{}
		u.SetGroupVersionKind(sch)

		err := c.List(context.Background(), u, &client.ListOptions{
			Namespace: "",
		})
		if err != nil {
			return err
		}

		output := ""
		backedUpObjects := 0
		for _, item := range u.Items {
			if item.Object == nil {
				continue
			}

			uid, ok, err := unstructured.NestedString(item.Object, "metadata", "uid")
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("object has no UID")
			}

			if _, ok := seenUIDs[uid]; ok {
				continue
			}
			seenUIDs[uid] = true

			unstructured.RemoveNestedField(item.Object, "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(item.Object, "metadata", "resourceVersion")
			unstructured.RemoveNestedField(item.Object, "metadata", "selfLink")
			unstructured.RemoveNestedField(item.Object, "metadata", "uid")
			unstructured.RemoveNestedField(item.Object, "status")

			marshalledItem, err := yaml.Marshal(item.Object)
			if err != nil {
				return err
			}
			output += string(marshalledItem)
			output += "\n---\n"
			backedUpObjects += 1
		}

		if backedUpObjects > 0 {
			filePath := path.Join(
				dir,
				fmt.Sprintf("%s.%s.%s.yaml", sch.Group, sch.Version, strings.ToLower(sch.Kind)),
			)
			if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
				return err
			}
			log.Printf(
				"Backed up %d %s/%s %s",
				backedUpObjects, sch.Group, sch.Version, strings.TrimSuffix(sch.Kind, "List"),
			)
		}
	}

	return nil
}
