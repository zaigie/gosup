package hook

import (
	"bufio"
	"fmt"

	"github.com/zaigie/gosup/process"
)

type MyProcessHook struct {
}

func (hook MyProcessHook) BeforeStart(ctx process.HookContext) {
	fmt.Println("BeforeStart")
}

func (hook MyProcessHook) AfterStart(ctx process.HookContext) {
	fmt.Println("AfterStart")
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

func (hook MyProcessHook) AfterWait(ctx process.HookContext, err error) {
	if err != nil {
		fmt.Println("AfterWait error:", err)
	} else {
		fmt.Println("AfterWait")
	}
}
