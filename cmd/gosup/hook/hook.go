package hook

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/zaigie/gosup"
)

// MyProcessHook is a custom implementation of ProcessHook.
type MyProcessHook struct{}

func (hook MyProcessHook) BeforeStart(ctx gosup.HookContext) {
	fmt.Fprintf(os.Stdout, "[%s] Process[%s] starting with command: %s %s\n", ctx.Params["prefix"], ctx.ProcessID, ctx.ProcessName, strings.Join(ctx.ProcessArgs, " "))
}

func (hook MyProcessHook) AfterStart(ctx gosup.HookContext) {
	fmt.Fprintf(os.Stdout, "[%s] Process[%s] started\n", ctx.Params["prefix"], ctx.ProcessID)
}

func (hook MyProcessHook) BeforeWait(ctx gosup.HookContext) {
	go func() {
		scanner := bufio.NewScanner(ctx.Stdout)
		for scanner.Scan() {
			fmt.Fprintf(os.Stdout, "[%s] Process[%s]: %s\n", ctx.Params["prefix"], ctx.ProcessID, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			if strings.Contains(err.Error(), "file already closed") {
				return
			}
			fmt.Fprintf(os.Stderr, "[%s] Process[%s] error reading stdout: %v\n", ctx.Params["prefix"], ctx.ProcessID, err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(ctx.Stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "[%s] Process[%s]: %s\n", ctx.Params["prefix"], ctx.ProcessID, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			if strings.Contains(err.Error(), "file already closed") {
				return
			}
			fmt.Fprintf(os.Stderr, "[%s] Process[%s] error reading stderr: %v\n", ctx.Params["prefix"], ctx.ProcessID, err)
		}
	}()
}

func (hook MyProcessHook) AfterWait(ctx gosup.HookContext, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Process[%s] wait error: %v\n", ctx.Params["prefix"], ctx.ProcessID, err)
	} else {
		fmt.Fprintf(os.Stdout, "[%s] Process[%s] done\n", ctx.Params["prefix"], ctx.ProcessID)
	}
}
