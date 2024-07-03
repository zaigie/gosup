package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/zaigie/gosup/cmd/gosup/hook"
	"github.com/zaigie/gosup/process"
)

func main() {
	pm := process.NewManager()
	defer pm.KillAll()

	wd, _ := os.Getwd()
	scriptPath := filepath.Join(wd, "test/run.py")
	cmdArgs := []string{"-u", scriptPath}
	p1, err := pm.Start("python", cmdArgs, hook.MyProcessHook{}, map[string]interface{}{
		"prefix": "gosup",
	})
	if err != nil {
		fmt.Println(err)
	}

	time.Sleep(2 * time.Second)

	_, err = pm.StartWithID("p2", "python", cmdArgs, nil, nil)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("System: waiting for 8 seconds\n")
	time.Sleep(8 * time.Second)

	pm.Stop(p1)
	err = pm.StopWithSignal("p2", syscall.SIGTERM)
	if err != nil {
		fmt.Println(err)
		pm.Stop("p2")
	}

	pm.WaitAll()
	fmt.Printf("System: all processes are done\n")
}
