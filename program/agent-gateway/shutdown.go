package main

import (
    "context"
    "errors"
    "fmt"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "time"
)

func logShutdownAuditFailure() {
    log.Print("shutdown_request_audit_failed: proceeding with local shutdown")
}

func (g *Gateway) requestShutdown(w http.ResponseWriter,r *http.Request) {
    host,_,err:=net.SplitHostPort(r.RemoteAddr)
    if err!=nil||!net.ParseIP(host).IsLoopback() { fail(w,403,"local_only");return }
    if r.Method!="POST" { fail(w,405,"method_not_allowed");return }
    if r.URL.RawQuery!="" { fail(w,400,"unsupported_query");return }
    body,err:=io.ReadAll(http.MaxBytesReader(w,r.Body,1))
    if err!=nil||len(body)!=0 { fail(w,400,"invalid_body");return }
    if g.shutdown==nil { fail(w,503,"shutdown_unavailable");return }
    writeJSON(w,http.StatusAccepted,map[string]string{"status":"shutting_down"})
    g.shutdown()
}

func shutdownAddress(listen string) (string,error) {
    host,port,err:=net.SplitHostPort(listen)
    if err!=nil { return "",errors.New("invalid gateway listen address") }
    ip:=net.ParseIP(host)
    number,err:=strconv.Atoi(port)
    if ip==nil||err!=nil||number<1||number>65535 { return "",errors.New("invalid gateway listen address") }
    if ip.IsUnspecified() {
        if ip.To4()!=nil { host="127.0.0.1" } else { host="::1" }
    } else if !ip.IsLoopback() {
        return "",errors.New("stop requires loopback or wildcard listening; use the server console for this configuration")
    }
    return net.JoinHostPort(host,port),nil
}

func lockReleased(path string) (bool,error) {
    _,err:=os.Stat(path)
    if os.IsNotExist(err) { return true,nil }
    if err!=nil { return false,errors.New("cannot inspect gateway state lock") }
    return false,nil
}

func stopGateway(path string) error {
    // Stopping must not depend on an available backend executable or open the store.
    data,err:=os.ReadFile(path)
    if err!=nil||len(data)>maxBody { return errors.New("cannot read gateway stop configuration") }
    var c Config
    if strictDecode(data,&c)!=nil { return errors.New("invalid gateway stop configuration") }
    if c.Listen==""&&c.Mode!="shared" { c.Listen="127.0.0.1:43127" }
    address,err:=shutdownAddress(c.Listen)
    if err!=nil { return err }
	if c.BackendMode=="" { c.BackendMode="codex" }
	if c.BackendMode!="codex"&&c.BackendMode!="openai_proxy" { return errors.New("invalid backend_mode") }
	if c.BackendMode=="codex"&&!filepath.IsAbs(c.StateDir) { return errors.New("gateway state_dir must be absolute") }
    key:=""
    for _,p:=range c.Principals {
        if p.Enabled { for _,candidate:=range p.Keys { if validKey(candidate) { key=candidate;break } } }
        if key!="" { break }
    }
    if key=="" { return errors.New("no enabled gateway credential configured") }
	lockPath:=""
	if c.BackendMode=="codex" { lockPath=filepath.Join(c.StateDir,"gateway.lock") }
    transport:=&http.Transport{DialContext:(&net.Dialer{Timeout:3*time.Second}).DialContext}
    defer transport.CloseIdleConnections()
    client:=&http.Client{Transport:transport,Timeout:5*time.Second,CheckRedirect:func(*http.Request,[]*http.Request) error { return errors.New("redirect rejected") }}
    request,err:=http.NewRequestWithContext(context.Background(),"POST","http://"+address+"/internal/shutdown",nil)
    if err!=nil { return errors.New("cannot prepare gateway stop request") }
    request.Header.Set("Authorization","Bearer "+key)
    response,err:=client.Do(request)
    if err!=nil {
		released,lockErr:=true,error(nil)
		if lockPath!="" { released,lockErr=lockReleased(lockPath) }
		if lockErr==nil&&released&&connectionRefused(err) {
            fmt.Println("Gateway is already stopped.");return nil
        }
        return errors.New("gateway stop request failed; no process was force-killed and no lock was removed")
    }
    body,readErr:=io.ReadAll(io.LimitReader(response.Body,1024))
    response.Body.Close()
    if response.StatusCode==http.StatusNotFound {
        return errors.New("running gateway does not support stop; rebuild and restart it after closing its old console")
    }
    var reply struct { Status string `json:"status"` }
    if response.StatusCode!=http.StatusAccepted||readErr!=nil||strictDecode(body,&reply)!=nil||reply.Status!="shutting_down" {
        return errors.New("gateway rejected the stop request")
    }
    deadline:=time.Now().Add(15*time.Second)
    for time.Now().Before(deadline) {
		released:=true
		if lockPath!="" { var inspectErr error;released,inspectErr=lockReleased(lockPath);if inspectErr!=nil { return inspectErr } }
        connection,dialErr:=net.DialTimeout("tcp",address,200*time.Millisecond)
        if dialErr==nil { connection.Close() }
        if released&&connectionRefused(dialErr) {
			fmt.Println("Gateway stopped gracefully.");return nil
        }
        time.Sleep(100*time.Millisecond)
    }
    return errors.New("gateway shutdown did not finish within 15 seconds; inspect the server log and preserve any remaining state lock")
}
