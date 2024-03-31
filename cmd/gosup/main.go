package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/zaigie/gosup/process"
)

type MyProcessHook struct {
}

func (hook MyProcessHook) BeforeWait(ctx process.HookContext) {
	go func() {
		scanner := bufio.NewScanner(ctx.Stdout)
		for scanner.Scan() {
			fmt.Printf("[%s]STDOUT: %s\n", ctx.Params["prefix"], scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(ctx.Stderr)
		for scanner.Scan() {
			fmt.Printf("[%s]STDERR: %s\n", ctx.Params["prefix"], scanner.Text())
		}
	}()
}

func main() {
	pm := process.NewManager()
	// defer pm.KillAll()

	wd, _ := os.Getwd()
	scriptPath := filepath.Join(wd, "test/run.py")
	args1 := []string{"-u", scriptPath}
	hook := MyProcessHook{}
	hookParams := map[string]interface{}{
		"prefix": "hello",
	}
	_, err := pm.Start("python", args1, hook, hookParams)
	if err != nil {
		fmt.Println(err)
	}

	time.Sleep(2 * time.Second)

	args2 := []string{"-u", scriptPath}
	_, err = pm.Start("python", args2, nil, nil)
	if err != nil {
		fmt.Println(err)
	}

	log.Println("waiting for 8 seconds")
	time.Sleep(20 * time.Second)
	pm.KillAll()
	log.Println("killed all processes")

	pm.WaitAll()
	log.Println("all processes are done")
}
