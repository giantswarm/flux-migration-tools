package main

import (
	"context"
	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	appSchema = schema.GroupVersionKind{Group: "kustomize.toolkit.fluxcd.io", Kind: "KustomizationList", Version: "v1beta2"}
)

func main() {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Fatal("failed to create client")
	}

	u := &unstructured.UnstructuredList{}
	u.SetGroupVersionKind(appSchema)

	err = c.List(context.Background(), u, &client.ListOptions{
		Namespace: "flux-giantswarm",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Clearing status in %d Flux Kustomizations...", len(u.Items))

	patch := []byte(`{
"op": "replace",
"path": "/status",
"value": {}
}`)
	for i := range u.Items {
		ptr := &u.Items[i]
		err = c.Status().Patch(
			context.Background(),
			ptr,
			client.RawPatch(types.JSONPatchType, patch),
		)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("  %s/%s", ptr.GetNamespace(), ptr.GetName())
	}
	log.Println("DONE")
}
