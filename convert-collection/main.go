package main

import (
	"log"
	"os"
	"path"
	"strings"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/yaml"
)

var (
	collectionDir string
	outputDir     string
)

type Konfigure struct {
	ApiVersion string        `json:"api_version"`
	Kind       string        `json:"kind"`
	Metadata   KonfigureMeta `json:"metadata"`

	AppCatalog              string `json:"app_catalog"`
	AppDestinationNamespace string `json:"app_destination_namespace"`
	AppDisableForceUpdate   bool   `json:"app_disable_force_update,omitempty"`
	AppName                 string `json:"app_name"`
	AppVersion              string `json:"app_version"`
	Name                    string `json:"name"`
}

type KonfigureMeta struct {
	Name        string            `json:"name,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Kustomization struct {
	Generators []string `json:"generators,omitempty"`
}

type pluginSetting struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func init() {
	flag.StringVar(&collectionDir, "collection-dir", "", "path to a directory containing the collection")
	flag.StringVar(&outputDir, "dir", "manifests", "target directory, inside collections directory, where all new resources will be saved")
	flag.Parse()
}

func main() {
	log.Println("Converting Argo Applications to Flux Kustomizations...")

	if collectionDir == "" {
		log.Fatal("collection-dir flag is required")
	}
	if outputDir == "" {
		log.Fatal("dir flag cannot be empty")
	}

	files, err := os.ReadDir(path.Join(collectionDir, "manifests"))
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(path.Join(collectionDir, outputDir), 0755); err != nil {
		log.Fatal(err)
	}

	kustomization := &Kustomization{
		Generators: []string{},
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}
		log.Println(file.Name())
		if err := convert(file.Name()); err != nil {
			log.Fatal(err)
		}
		kustomization.Generators = append(kustomization.Generators, file.Name())
	}

	if err := saveKustomization(kustomization); err != nil {
		log.Fatal(err)
	}
	log.Println("DONE")
}

func convert(fileName string) error {
	var app = &unstructured.Unstructured{}
	{
		// load Argo Application CR
		b, err := os.ReadFile(path.Join(collectionDir, "manifests", fileName))
		if err != nil {
			return err
		}

		app.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "argoproj.io",
			Kind:    "Application",
			Version: "v1alpha1",
		})

		err = yaml.Unmarshal(b, &app.Object)
		if err != nil {
			return err
		}
	}

	var generator *Konfigure
	{
		// Create a generator matching Argo Application and populate it
		generator = &Konfigure{
			ApiVersion: "generators.giantswarm.io/v1",
			Kind:       "Konfigure",
			Metadata: KonfigureMeta{
				Name: app.GetName(),
				Annotations: map[string]string{
					"config.kubernetes.io/function": "exec:\n  path: /plugins/konfigure",
				},
			},
			AppDisableForceUpdate: false,
		}

		destinationNamespace, ok, err := unstructured.NestedString(app.Object, "spec", "destination", "namespace")
		if err != nil {
			return err
		}
		if !ok {
			log.Fatalf("Destination namespace not found in %s", fileName)
		}
		generator.AppDestinationNamespace = destinationNamespace

		pluginConfigI, ok, err := unstructured.NestedFieldNoCopy(app.Object, "spec", "source", "plugin", "env")
		if err != nil {
			return err
		}
		if !ok {
			log.Fatalf("Plugin configuration not found in %s", fileName)
		}

		var pluginSettings []pluginSetting
		b, err := yaml.Marshal(pluginConfigI)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(b, &pluginSettings); err != nil {
			return err
		}

		for _, setting := range pluginSettings {
			switch setting.Name {
			case "KONFIGURE_APP_NAME":
				generator.AppName = setting.Value
				generator.Name = setting.Value
			case "KONFIGURE_APP_VERSION":
				generator.AppVersion = setting.Value
			case "KONFIGURE_APP_CATALOG":
				generator.AppCatalog = setting.Value
			default:
				log.Fatalf("Unexpected plugin setting %q", setting.Name)
			}
		}
	}

	// marshal and save the generator
	b, err := yaml.Marshal(generator)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(collectionDir, outputDir, fileName), b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func saveKustomization(k *Kustomization) error {
	b, err := yaml.Marshal(k)
	if err != nil {
		return err
	}

	return os.WriteFile(path.Join(collectionDir, outputDir, "kustomization.yaml"), b, 0644)
}
