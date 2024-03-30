package main

import (
	"fmt"

	"github.com/zaigie/gosup/process"
)

func main() {
	pm := process.NewManager()
	defer pm.KillAll()

	args1 := []string{"-c", "echo hello"}
	_, err := pm.Start("sh", args1, nil)
	if err != nil {
		fmt.Println(err)
	}
	args2 := []string{"-c", "sleep 5 && echo world"}
	pm.Start("sh", args2, nil)

	pm.WaitAll()
}
