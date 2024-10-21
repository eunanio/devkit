package main

import (
	"github.com/eunanio/devkit/pkg/log"
	"github.com/eunanio/devkit/pkg/system"
)

func main() {
	err := system.OpenURL("http://google.com")
	log.NoError(err, "failed to open url")
}
