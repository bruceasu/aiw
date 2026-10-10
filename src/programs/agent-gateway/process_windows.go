package main

import (
    "errors"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "syscall"
    "time"
    "unsafe"
)

var kernel32=syscall.NewLazyDLL("kernel32.dll")
var createJob=kernel32.NewProc("CreateJobObjectW")
var setJob=kernel32.NewProc("SetInformationJobObject")
var assignJob=kernel32.NewProc("AssignProcessToJobObject")
var terminateJob=kernel32.NewProc("TerminateJobObject")
var resumeThread=kernel32.NewProc("ResumeThread")
var initializeAttributes=kernel32.NewProc("InitializeProcThreadAttributeList")
var updateAttributes=kernel32.NewProc("UpdateProcThreadAttribute")
var deleteAttributes=kernel32.NewProc("DeleteProcThreadAttributeList")
var createNativeProcess=kernel32.NewProc("CreateProcessW")
var queryJob=kernel32.NewProc("QueryInformationJobObject")

type basicJobLimits struct {
    PerProcessUserTime int64
    PerJobUserTime int64
    LimitFlags uint32
    MinimumWorkingSet uintptr
    MaximumWorkingSet uintptr
    ActiveProcessLimit uint32
    Affinity uintptr
    PriorityClass uint32
    SchedulingClass uint32
}
type extendedJobLimits struct {
    Basic basicJobLimits
    IO [6]uint64
    ProcessMemoryLimit uintptr
    JobMemoryLimit uintptr
    PeakProcessMemoryUsed uintptr
    PeakJobMemoryUsed uintptr
}
type startupInfoEx struct {
    Info syscall.StartupInfo
    Attributes uintptr
}
type process struct {
    handle syscall.Handle
    job syscall.Handle
    done chan error
}

func validateExecutable(c Config) error {
    if !strings.EqualFold(filepath.Ext(c.CodexPath),".exe") { return errors.New("Windows requires a real exe; use node.exe and codex_script for npm installs") }
    if c.CodexScript!=""&&!strings.EqualFold(filepath.Ext(c.CodexScript),".js") { return errors.New("codex_script must be a JavaScript entry") }
    return nil
}

func startProcess(path string,args,env []string,cwd string,stdin,stdout,stderr *os.File) (*process,error) {
    job,_,err:=createJob.Call(0,0)
    if job==0 { return nil,err }
    success:=false
    defer func(){if !success { syscall.CloseHandle(syscall.Handle(job)) }}()
    limits:=extendedJobLimits{}
    limits.Basic.LimitFlags=0x2000 // JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
    ok,_,err:=setJob.Call(job,9,uintptr(unsafe.Pointer(&limits)),unsafe.Sizeof(limits))
    if ok==0 { return nil,err }
    handles:=[]syscall.Handle{syscall.Handle(stdin.Fd()),syscall.Handle(stdout.Fd()),syscall.Handle(stderr.Fd())}
    defer func(){for _,handle:=range handles { _=syscall.SetHandleInformation(handle,syscall.HANDLE_FLAG_INHERIT,0) }}()
    for _,handle:=range handles {
        if err=syscall.SetHandleInformation(handle,syscall.HANDLE_FLAG_INHERIT,syscall.HANDLE_FLAG_INHERIT);err!=nil { return nil,err }
    }
    var size uintptr
    initializeAttributes.Call(0,1,0,uintptr(unsafe.Pointer(&size)))
    if size==0||size>65536 { return nil,errors.New("cannot allocate process attributes") }
    attributes:=make([]byte,size)
    ptr:=uintptr(unsafe.Pointer(&attributes[0]))
    ok,_,err=initializeAttributes.Call(ptr,1,0,uintptr(unsafe.Pointer(&size)))
    if ok==0 { return nil,err }
    defer func(){deleteAttributes.Call(ptr);runtime.KeepAlive(attributes)}()
    ok,_,err=updateAttributes.Call(ptr,0,0x20002,uintptr(unsafe.Pointer(&handles[0])),uintptr(len(handles))*unsafe.Sizeof(handles[0]),0,0)
    if ok==0 { return nil,err }
    si:=startupInfoEx{}
    si.Info.Cb=uint32(unsafe.Sizeof(si));si.Info.Flags=0x100;si.Info.StdInput=handles[0];si.Info.StdOutput=handles[1];si.Info.StdErr=handles[2];si.Attributes=ptr
    command:=syscall.EscapeArg(path)
    for _,arg:=range args { command+=" "+syscall.EscapeArg(arg) }
    application,e:=syscall.UTF16PtrFromString(path);if e!=nil { return nil,e }
    commandLine,e:=syscall.UTF16PtrFromString(command);if e!=nil { return nil,e }
    directory,e:=syscall.UTF16PtrFromString(cwd);if e!=nil { return nil,e }
    // CreateProcessW requires a double NUL terminated Unicode environment block.
    block:=[]uint16{}
    for _,entry:=range env { encoded,e:=syscall.UTF16FromString(entry);if e!=nil { return nil,e };block=append(block,encoded...) }
    block=append(block,0);if len(block)==1 { block=append(block,0) }
    pi:=syscall.ProcessInformation{}
    const flags=0x4|0x400|0x80000|0x08000000 // SUSPENDED | UNICODE_ENV | EXTENDED_STARTUPINFO | NO_WINDOW
    ok,_,err=createNativeProcess.Call(uintptr(unsafe.Pointer(application)),uintptr(unsafe.Pointer(commandLine)),0,0,1,flags,uintptr(unsafe.Pointer(&block[0])),uintptr(unsafe.Pointer(directory)),uintptr(unsafe.Pointer(&si)),uintptr(unsafe.Pointer(&pi)))
    if ok==0 { return nil,err }
    defer syscall.CloseHandle(pi.Thread)
    ok,_,err=assignJob.Call(job,uintptr(pi.Process))
    if ok==0 { _=syscall.TerminateProcess(pi.Process,1);syscall.CloseHandle(pi.Process);return nil,err }
    resumed,_,err:=resumeThread.Call(uintptr(pi.Thread))
    if resumed==0xffffffff { terminateJob.Call(job,1);syscall.CloseHandle(pi.Process);return nil,err }
    p:=&process{handle:pi.Process,job:syscall.Handle(job),done:make(chan error,1)}
    success=true
    go func(){
        _,waitErr:=syscall.WaitForSingleObject(p.handle,syscall.INFINITE)
        var exit uint32
        if waitErr==nil { waitErr=syscall.GetExitCodeProcess(p.handle,&exit) }
        if waitErr==nil&&exit!=0 { waitErr=errors.New("backend exited unsuccessfully") }
        p.done<-waitErr
    }()
    return p,nil
}

func (p *process) stopTree() error {
    ok,_,err:=terminateJob.Call(uintptr(p.job),1)
    if ok==0 { return err }
    // The Job termination applies to every member, including descendants.
    deadline:=time.Now().Add(5*time.Second)
    for time.Now().Before(deadline) {
        var accounting struct { Times [4]int64; Faults uint32; Total uint32; Active uint32; Terminated uint32 }
        ok,_,err=queryJob.Call(uintptr(p.job),1,uintptr(unsafe.Pointer(&accounting)),unsafe.Sizeof(accounting),0)
        if ok==0 { return err };if accounting.Active==0 { return nil }
        time.Sleep(25*time.Millisecond)
    }
    return errors.New("process did not exit within cleanup deadline")
}

func (p *process) close() { syscall.CloseHandle(p.handle);syscall.CloseHandle(p.job) }
