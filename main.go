package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/kosli-dev/terraform-provider-kosli/internal/provider"
)

// version is set via ldflags during the build process.
// See GoReleaser configuration for details.
var version = "dev"

// commit is set via ldflags during the build process.
var commit = "none" //nolint:unused // Set via ldflags

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/kosli-dev/kosli",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
