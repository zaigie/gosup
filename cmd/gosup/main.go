package main

import (
	"fmt"
	"log"
	"time"

	"github.com/zaigie/gosup/process"
)

func main() {
	pm := process.NewManager()
	// defer pm.KillAll()

	args1 := []string{"-c", "sleep 3 && echo hello"}
	_, err := pm.Start("sh", args1, nil)
	if err != nil {
		fmt.Println(err)
	}
	args2 := []string{"-c", "sleep 5 && echo world"}
	pm.Start("sh", args2, nil)

	log.Println("waiting for 2 seconds")
	time.Sleep(2 * time.Second)
	pm.KillAll()
	log.Println("killed all processes")

	pm.WaitAll()
	log.Println("all processes are done")
}
