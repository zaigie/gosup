package process

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
)

var (
	ErrorProcessNotFound = fmt.Errorf("process not found")
	ErrorProcessRunning  = fmt.Errorf("process is already running")
)

type ProcessManager struct {
	mu      sync.RWMutex
	Process map[int]*Process
	nextID  int
	wg      sync.WaitGroup
}

func NewManager() *ProcessManager {
	return &ProcessManager{
		Process: make(map[int]*Process),
	}
}

type Process struct {
	Cmd    *exec.Cmd
	Before func(stdout, stderr io.ReadCloser)
	Stdout io.ReadCloser
	Stderr io.ReadCloser
}

func (pm *ProcessManager) List() []int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	ids := make([]int, 0, len(pm.Process))
	for id := range pm.Process {
		ids = append(ids, id)
	}

	return ids
}

func (pm *ProcessManager) Get(id int) (*Process, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	process, exists := pm.Process[id]
	if !exists {
		return nil, ErrorProcessNotFound
	}

	return process, nil
}

func (pm *ProcessManager) Start(name string, args []string, beforeWait func(stdout, stderr io.ReadCloser)) (int, error) {
	pm.mu.Lock()
	id := pm.nextID
	pm.nextID++
	pm.mu.Unlock()

	cmd := exec.Command(name, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	pm.mu.Lock()
	pm.Process[id] = &Process{Cmd: cmd, Before: beforeWait, Stdout: stdoutPipe, Stderr: stderrPipe}
	pm.wg.Add(1)
	pm.mu.Unlock()

	go func() {
		defer stdoutPipe.Close()
		defer stderrPipe.Close()
		if beforeWait != nil {
			beforeWait(stdoutPipe, stderrPipe)
		} else {
			handlePipe(id, stdoutPipe, stderrPipe)
		}

		cmd.Wait()

		pm.mu.Lock()
		delete(pm.Process, id)
		pm.mu.Unlock()

		pm.wg.Done()
	}()

	return id, nil
}

func (pm *ProcessManager) Stop(id int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	process, exists := pm.Process[id]
	if !exists {
		return ErrorProcessNotFound
	}

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
		if err := process.Cmd.Process.Kill(); err != nil {
			// log.Printf("failed to kill process %d: %v\n", id, err)
			errs = append(errs, fmt.Errorf("failed to kill process %d: %v", id, err))
			continue
		}
		delete(pm.Process, id)
	}
	return errs
}

func (pm *ProcessManager) WaitAll() {
	pm.wg.Wait()
}

func handleScanner(id int, scanner *bufio.Scanner) {
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[%d]: %s\n", id, line)
	}
	if err := scanner.Err(); err != nil {
		if strings.Contains(err.Error(), "file already closed") {
			return
		}
		fmt.Printf("[%d] error reading: %v\n", id, err)
	}
}

func handlePipe(id int, stdout, stderr io.ReadCloser) {
	outScanner := bufio.NewScanner(stdout)
	handleScanner(id, outScanner)

	errScanner := bufio.NewScanner(stderr)
	handleScanner(id, errScanner)
}
