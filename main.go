package main

import (
	"os"

	"github.com/sapcc/concourse-netbox-resource/internal/resource"
	"github.com/tbe/resource-framework/log"
	fr "github.com/tbe/resource-framework/resource"
)

func main() {
	r := resource.NewNetboxResource()
	handler, err := fr.NewHandler(r)
	if err != nil {
		log.Error("error creating handler: %s", err)
		os.Exit(1)
	}
	_ = handler.Run()
}
