package main

import (
    "errors"
    "os"
    "os/exec"
    "syscall"
    "time"
)

type process struct {
    cmd *exec.Cmd
    done chan error
}

func validateExecutable(c Config) error {
    info,err:=os.Stat(c.CodexPath)
    if err!=nil||info.Mode()&0111==0 { return errors.New("codex path is not executable") }
    return nil
}

func startProcess(path string,args,env []string,cwd string,stdin,stdout,stderr *os.File) (*process,error) {
    cmd:=exec.Command(path,args...)
    cmd.Dir=cwd;cmd.Env=env;cmd.Stdin=stdin;cmd.Stdout=stdout;cmd.Stderr=stderr
    cmd.SysProcAttr=&syscall.SysProcAttr{Setpgid:true}
    if err:=cmd.Start();err!=nil { return nil,err }
    p:=&process{cmd:cmd,done:make(chan error,1)}
    go func(){p.done<-cmd.Wait()}()
    return p,nil
}

func (p *process) stopTree() error {
    group:=-p.cmd.Process.Pid
    err:=syscall.Kill(group,syscall.SIGTERM)
    if errors.Is(err,syscall.ESRCH) { return nil };if err!=nil { return err }
    deadline:=time.Now().Add(2*time.Second)
    for time.Now().Before(deadline) {
        if errors.Is(syscall.Kill(group,0),syscall.ESRCH) { return nil };time.Sleep(25*time.Millisecond)
    }
    if err=syscall.Kill(group,syscall.SIGKILL);err!=nil&&!errors.Is(err,syscall.ESRCH) { return err }
    deadline=time.Now().Add(3*time.Second)
    for time.Now().Before(deadline) {
        if errors.Is(syscall.Kill(group,0),syscall.ESRCH) { return nil };time.Sleep(25*time.Millisecond)
    }
    return errors.New("process group did not disappear within cleanup deadline")
}

func (p *process) close() {}
