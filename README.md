# gosup

Simply manage the start, stop and output of multiple processes through os/exec

## Quick Start

```go
func main() {
	pm := gosup.NewManager()
	defer pm.KillAll()

    wd, _ := os.Getwd()
	scriptPath := filepath.Join(wd, "test/run.py")

    p1, err := pm.Start("python", []string{"-u", scriptPath}, nil, nil)
    if err != nil {
		fmt.Println(err)
        return
	}

    time.Sleep(8 * time.Second)
    pm.Stop(p1)
    pm.WaitAll()
}
```

## Process Manage

(Wait for completion)

## Process Hook

(Wait for completion)
