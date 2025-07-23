package main

import (
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate go run generic_generator_impl.go
type GeneratorType struct {
	TypeName string
}

func main() {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	kinds := scheme.KnownTypes(v1alpha1.SchemeGroupVersion)
	generators := make([]GeneratorType, 0, len(kinds))

	for kind := range kinds {
		// Ignore kubernetes default types and generators lists
		if strings.HasSuffix(kind, "List") ||
			kind == "CreateOptions" ||
			kind == "DeleteOptions" ||
			kind == "GetOptions" ||
			kind == "ListOptions" ||
			kind == "PatchOptions" ||
			kind == "UpdateOptions" ||
			kind == "WatchEvent" ||
			kind == "GeneratorState" {
			continue
		}

		generators = append(generators, GeneratorType{TypeName: kind})
	}

	headerTmpl := template.Must(template.ParseFiles("gen_generic_generator_header.gotmpl"))
	bodyTmpl := template.Must(template.ParseFiles("gen_generic_generator_body.gotmpl"))

	outFile, err := os.Create("../zz_generated_generic_generators.go")
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Render header once
	if err := headerTmpl.Execute(outFile, nil); err != nil {
		log.Fatalf("failed to execute header template: %v", err)
	}

	// Render each generator body
	for _, g := range generators {
		if err := bodyTmpl.Execute(outFile, g); err != nil {
			log.Fatalf("failed to execute body template: %v", err)
		}
	}
}
