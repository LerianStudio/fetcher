package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	in "github.com/LerianStudio/fetcher/v2/components/manager/internal/adapters/http/in"
)

func main() {
	output := flag.String("output", "components/manager/api/openapi.yaml", "OpenAPI YAML output path, or - for stdout")

	flag.Parse()

	if *output == "-" {
		spec, err := in.GenerateCanonicalSpec()
		if err != nil {
			log.Fatal(err)
		}

		if _, err := os.Stdout.Write(spec); err != nil {
			log.Fatal(err)
		}

		return
	}

	if err := writeSpec(*output); err != nil {
		log.Fatal(err)
	}
}

func writeSpec(path string) error {
	spec, err := in.GenerateCanonicalSpec()
	if err != nil {
		return fmt.Errorf("generate canonical OpenAPI spec: %w", err)
	}

	// #nosec G306 -- api/openapi.yaml is a public, versioned artifact.
	if err := os.WriteFile(path, spec, 0o644); err != nil {
		return fmt.Errorf("write OpenAPI spec %s: %w", path, err)
	}

	return nil
}
