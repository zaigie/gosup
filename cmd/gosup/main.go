package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/zaigie/gosup/cmd/gosup/hook"
	"github.com/zaigie/gosup/process"
)

func main() {
	pm := process.NewManager()
	// defer pm.KillAll()

	wd, _ := os.Getwd()
	scriptPath := filepath.Join(wd, "test/run.py")
	args1 := []string{"-u", scriptPath}
	_, err := pm.Start("", "python", args1, hook.MyProcessHook{}, map[string]interface{}{
		"prefix": "hello",
	})
	if err != nil {
		fmt.Println(err)
	}

	time.Sleep(2 * time.Second)

	// args2 := []string{"-u", scriptPath}
	// _, err = pm.Start("python", args2, nil, nil)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	log.Println("waiting for 8 seconds")
	time.Sleep(8 * time.Second)
	pm.KillAll()
	log.Println("killed all processes")

	pm.WaitAll()
	log.Println("all processes are done")
}
