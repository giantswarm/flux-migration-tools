package main

import (
	"context"
	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	appSchema = schema.GroupVersionKind{Group: "argoproj.io", Kind: "ApplicationList", Version: "v1alpha1"}
)

func main() {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Fatal("failed to create client")
	}

	u := &unstructured.UnstructuredList{}
	u.SetGroupVersionKind(appSchema)

	err = c.List(context.Background(), u, &client.ListOptions{
		Namespace: "",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Pausing %d Argo Applications...", len(u.Items))

	patch := []byte(`{
"op": "replace",
"path": "/spec/syncPolicy/automated/selfHeal",
"value": false
}`)
	for i := range u.Items {
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
	log.Println("DONE")
}
