package gosup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
)

var (
	ErrorProcessNotFound = fmt.Errorf("process not found")
	ErrorProcessRunning  = fmt.Errorf("process is already running")
)

/* ProcessManager */

// ProcessManager is a manager for processes.
type ProcessManager struct {
	mu        sync.RWMutex
	Processes map[string]*Process
	wg        sync.WaitGroup
}

// NewManager creates a new ProcessManager.
func NewManager() *ProcessManager {
	return &ProcessManager{
		Processes: make(map[string]*Process),
	}
}

// Process is a running process.
type Process struct {
	Cmd      *exec.Cmd
	Stdout   io.ReadCloser
	Stderr   io.ReadCloser
	stopChan chan struct{}
}

// List returns the list of process IDs.
func (pm *ProcessManager) List() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	ids := make([]string, 0, len(pm.Processes))
	for id := range pm.Processes {
		ids = append(ids, id)
	}

	return ids
}

// Get returns the process with the given ID.
func (pm *ProcessManager) Get(id string) (*Process, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	process, exists := pm.Processes[id]
	if !exists {
		return nil, ErrorProcessNotFound
	}

	return process, nil
}

// IsRunning returns true if the process with the given ID is running.
func (pm *ProcessManager) IsRunning(id string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, exists := pm.Processes[id]
	return exists
}

// StartWithID starts a process with the given ID.
func (pm *ProcessManager) StartWithID(id string, name string, args []string, hook ProcessHook, hookParams map[string]interface{}) (string, error) {
	if process, _ := pm.Get(id); process != nil {
		return "", ErrorProcessRunning
	}

	cmd := exec.Command(name, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if hook == nil {
		hook = DefaultProcessHook{}
	}

	hookCtx := HookContext{
		Stdout:      stdoutPipe,
		Stderr:      stderrPipe,
		Params:      hookParams,
		ProcessID:   id,
		ProcessName: name,
		ProcessArgs: args,
	}

	hook.BeforeStart(hookCtx)

	if err := cmd.Start(); err != nil {
		return "", err
	}

	pm.mu.Lock()
	pm.Processes[id] = &Process{Cmd: cmd, Stdout: stdoutPipe, Stderr: stderrPipe, stopChan: make(chan struct{})}
	pm.wg.Add(1)
	pm.mu.Unlock()

	hook.AfterStart(hookCtx)

	go func() {
		var err error
		defer stdoutPipe.Close()
		defer stderrPipe.Close()

		hook.BeforeWait(hookCtx)

		select {
		case <-pm.Processes[id].stopChan:

		case <-func() chan struct{} {
			ch := make(chan struct{})
			go func() {
				defer close(ch)
				err = cmd.Wait()
			}()
			return ch
		}():

		}

		pm.mu.Lock()
		delete(pm.Processes, id)
		pm.mu.Unlock()

		pm.wg.Done()

		if hook != nil {
			hook.AfterWait(hookCtx, err)
		}
	}()

	return id, nil
}

// Start starts a process with a random ID.
func (pm *ProcessManager) Start(name string, args []string, hook ProcessHook, hookParams map[string]interface{}) (string, error) {
	id := uuid.New().String()[:8]
	return pm.StartWithID(id, name, args, hook, hookParams)
}

func isStopSignal(sig syscall.Signal) bool {
	switch sig {
	case syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGKILL:
		return true
	}
	return false
}

// StopWithSignal stops the process with the given ID using the given signal.
func (pm *ProcessManager) StopWithSignal(id string, sig syscall.Signal) error {
	if !isStopSignal(sig) {
		return fmt.Errorf("invalid signal: %v, only SIGINT, SIGTERM, SIGHUP, SIGQUIT, and SIGKILL are allowed", sig)
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()

	process, exists := pm.Processes[id]
	if !exists {
		return ErrorProcessNotFound
	}

	close(process.stopChan)

	if err := process.Cmd.Process.Signal(sig); err != nil {
		return err
	}

	if sig != syscall.SIGKILL {
		go func() {
			process.Cmd.Wait()
			delete(pm.Processes, id)
		}()
	} else {
		delete(pm.Processes, id)
	}

	return nil
}

// Stop stops the process with the given ID.
func (pm *ProcessManager) Stop(id string) error {
	return pm.StopWithSignal(id, syscall.SIGKILL)
}

// KillAll kills all running processes.
func (pm *ProcessManager) KillAll() []error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	errs := make([]error, 0, len(pm.Processes))
	for id, process := range pm.Processes {
		close(process.stopChan)
		if err := process.Cmd.Process.Kill(); err != nil {
			errs = append(errs, fmt.Errorf("failed to kill process %s: %v", id, err))
			continue
		}
		delete(pm.Processes, id)
	}
	return errs
}

// WaitAll waits for all running processes to finish.
func (pm *ProcessManager) WaitAll() {
	pm.wg.Wait()
}

/* Hook */

// HookContext is the context passed to the ProcessHook methods.
type HookContext struct {
	Stdout      io.ReadCloser
	Stderr      io.ReadCloser
	Params      map[string]interface{}
	ProcessID   string
	ProcessName string
	ProcessArgs []string
}

// ProcessHook is the interface for process hooks.
type ProcessHook interface {
	// BeforeStart is called before the process is started.
	BeforeStart(ctx HookContext)
	// AfterStart is called after the process is started.
	AfterStart(ctx HookContext)
	// BeforeWait is called before the process is waited.
	BeforeWait(ctx HookContext)
	// AfterWait is called after the process is waited.
	AfterWait(ctx HookContext, err error)
}

// DefaultProcessHook is the default implementation of ProcessHook.
type DefaultProcessHook struct{}

func (hook DefaultProcessHook) BeforeStart(ctx HookContext) {
	fmt.Fprintf(os.Stdout, "Process[%s] starting with command: %s %s\n", ctx.ProcessID, ctx.ProcessName, strings.Join(ctx.ProcessArgs, " "))
}

func (hook DefaultProcessHook) AfterStart(ctx HookContext) {
	fmt.Fprintf(os.Stdout, "Process[%s] started\n", ctx.ProcessID)
}

func (hook DefaultProcessHook) BeforeWait(ctx HookContext) {
	go func() {
		scanner := bufio.NewScanner(ctx.Stdout)
		for scanner.Scan() {
			fmt.Fprintf(os.Stdout, "Process[%s]: %s\n", ctx.ProcessID, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			if strings.Contains(err.Error(), "file already closed") {
				return
			}
			fmt.Fprintf(os.Stderr, "Process[%s] error reading stdout: %v\n", ctx.ProcessID, err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(ctx.Stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "Process[%s]: %s\n", ctx.ProcessID, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			if strings.Contains(err.Error(), "file already closed") {
				return
			}
			fmt.Fprintf(os.Stderr, "Process[%s] error reading stderr: %v\n", ctx.ProcessID, err)
		}
	}()
}

func (hook DefaultProcessHook) AfterWait(ctx HookContext, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Process[%s] wait error: %v\n", ctx.ProcessID, err)
	} else {
		fmt.Fprintf(os.Stdout, "Process[%s] done\n", ctx.ProcessID)
	}
}
