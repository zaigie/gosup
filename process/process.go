package process

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrorProcessNotFound = fmt.Errorf("process not found")
	ErrorProcessRunning  = fmt.Errorf("process is already running")
)

type ProcessManager struct {
	mu      sync.RWMutex
	Process map[string]*Process
	wg      sync.WaitGroup
}

func NewManager() *ProcessManager {
	return &ProcessManager{
		Process: make(map[string]*Process),
	}
}

type HookContext struct {
	Stdout io.ReadCloser
	Stderr io.ReadCloser
	Params map[string]interface{}
}

type ProcessHook interface {
	BeforeStart(ctx HookContext)
	AfterStart(ctx HookContext)
	BeforeWait(ctx HookContext)
	AfterWait(ctx HookContext, err error)
}

type Process struct {
	Cmd      *exec.Cmd
	Stdout   io.ReadCloser
	Stderr   io.ReadCloser
	stopChan chan struct{}
}

func (pm *ProcessManager) List() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	ids := make([]string, 0, len(pm.Process))
	for id := range pm.Process {
		ids = append(ids, id)
	}

	return ids
}

func (pm *ProcessManager) Get(id string) (*Process, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	process, exists := pm.Process[id]
	if !exists {
		return nil, ErrorProcessNotFound
	}

	return process, nil
}

func (pm *ProcessManager) Start(id string, name string, args []string, hook ProcessHook, hookParams map[string]interface{}) (string, error) {
	pm.mu.Lock()
	if id == "" {
		id = uuid.New().String()
	}
	pm.mu.Unlock()

	cmd := exec.Command(name, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	pm.mu.Lock()
	pm.Process[id] = &Process{Cmd: cmd, Stdout: stdoutPipe, Stderr: stderrPipe, stopChan: make(chan struct{})}
	pm.wg.Add(1)
	pm.mu.Unlock()

	hookCtx := HookContext{
		Stdout: stdoutPipe,
		Stderr: stderrPipe,
		Params: hookParams,
	}

	if hook != nil {
		hook.BeforeStart(hookCtx)
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	if hook != nil {
		hook.AfterStart(hookCtx)
	}

	go func() {
		var err error
		defer stdoutPipe.Close()
		defer stderrPipe.Close()
		if hook != nil {
			hook.BeforeWait(hookCtx)
		} else {
			handlePipe(id, stdoutPipe, stderrPipe)
		}

		select {
		case <-pm.Process[id].stopChan:

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
		delete(pm.Process, id)
		pm.mu.Unlock()

		pm.wg.Done()

		if hook != nil {
			hook.AfterWait(hookCtx, err)
		}
	}()

	return id, nil
}

func (pm *ProcessManager) Stop(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	process, exists := pm.Process[id]
	if !exists {
		return ErrorProcessNotFound
	}

	close(process.stopChan)
	if err := process.Cmd.Process.Kill(); err != nil {
		return err
	}

	delete(pm.Process, id)
	return nil
}

func (pm *ProcessManager) KillAll() []error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	errs := make([]error, 0, len(pm.Process))
	for id, process := range pm.Process {
		close(process.stopChan)
		if err := process.Cmd.Process.Kill(); err != nil {
			// log.Printf("failed to kill process %d: %v\n", id, err)
			errs = append(errs, fmt.Errorf("failed to kill process %s: %v", id, err))
			continue
		}
		delete(pm.Process, id)
	}
	return errs
}

func (pm *ProcessManager) WaitAll() {
	pm.wg.Wait()
}

func handleScanner(id string, scanner *bufio.Scanner) {
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[%s]: %s\n", id, line)
	}
	if err := scanner.Err(); err != nil {
		if strings.Contains(err.Error(), "file already closed") {
			return
		}
		fmt.Printf("[%s] error reading: %v\n", id, err)
	}
}

func handlePipe(id string, stdout, stderr io.ReadCloser) {
	outScanner := bufio.NewScanner(stdout)
	handleScanner(id, outScanner)

	errScanner := bufio.NewScanner(stderr)
	handleScanner(id, errScanner)
}
