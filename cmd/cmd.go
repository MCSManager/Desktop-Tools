package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"mcsmanager.com/desktop-app/utils"
)

type ProcessMgr struct {
	Path     string
	Args     []string
	Started  bool
	stdin    io.WriteCloser
	startErr chan error
	exited   chan error
	cmder    *exec.Cmd
	Cwd      string
}

func NewProcessMgr(workDir string, path string, args ...string) *ProcessMgr {
	return &ProcessMgr{Path: path, Args: args, Cwd: workDir, startErr: make(chan error), exited: make(chan error)}
}

// ListenStop 监听程序停止运行
func (pm *ProcessMgr) ListenStop(callback func(err error)) {
	go func() {
		for {
			select {
			case err := <-pm.exited:
				pm.Started = false
				callback(err)
			}
		}
	}()
}

func (pm *ProcessMgr) Start() error {
	go pm.run()
	return <-pm.startErr
}

func (pm *ProcessMgr) run() {
	os.Chdir(pm.Cwd)
	fmt.Printf("Change CWD: %s\n", pm.Cwd)
	pm.cmder = exec.Command(pm.Path, pm.Args...)
	if runtime.GOOS == "windows" {
		pm.cmder.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	var err error
	pm.stdin, err = pm.cmder.StdinPipe()
	if err != nil {
		pm.startErr <- err
		return
	}
	err = pm.cmder.Start()
	pm.startErr <- err
	pm.Started = true
	pm.exited <- pm.cmder.Wait()
}

func (pm *ProcessMgr) End() error {
	if pm.cmder == nil || pm.stdin == nil {
		return nil
	}
	defer pm.stdin.Close()
	_, err := pm.stdin.Write([]byte("exit\n"))
	pm.ExitCheck()
	return err
}

func (pm *ProcessMgr) ExitCheck() {
	go func() {
		fmt.Println("Exit check enable..")
		time.Sleep(5 * time.Second)
		fmt.Printf("Program kill %v", pm.Started)
		if pm.Started {
			pid := pm.cmder.Process.Pid
			utils.WriteErrLog(fmt.Sprintf("Kill Program: taskkill /PID %d /T /F", pid))
			cmder := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
			cmder.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			err := cmder.Run()
			if err != nil {
				utils.WriteErrLog(fmt.Sprintf("Kill command Err: %s", err))
			}
		}
	}()
}
