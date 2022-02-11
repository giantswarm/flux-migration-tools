package main

import (
	"context"
	"log"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	schemas = []schema.GroupVersionKind{
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
			log.Println(err)
			continue
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
				log.Println(err)
				continue
			}
			log.Printf("  %s/%s", ptr.GetNamespace(), ptr.GetName())
		}
	}
	log.Println("DONE")
}
