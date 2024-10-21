package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/eunanio/devkit/pkg/log"
)

func OpenURL(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}

func GetStdin() (msg string) {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		msg = scanner.Text()
	}
	err := scanner.Err()
	log.NoError(err, "Error reading from stdin")

	return msg
}
