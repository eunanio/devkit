package main

import (
	"fmt"

	"github.com/eunanio/sdk/pkg/log"
	"github.com/eunanio/sdk/pkg/oci"
)

func main() {
	client := oci.NewOciClient()
	client.SetBasicAuth("username", "password")
	manifest, err := client.PullManifest(&oci.Tag{Host: "registry.hub.docker.com", Namespace: "library", Name: "nginx", Version: "latest"})
	log.NoError(err, "failed to push blob")
	fmt.Println(manifest)
}
