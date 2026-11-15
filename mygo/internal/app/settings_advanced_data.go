package app

import "os/exec"

func openURL(url string) {
	_ = exec.Command("open", url).Start()
}
