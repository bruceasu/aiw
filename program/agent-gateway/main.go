package main

import (
    "context"
    "errors"
    "flag"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func parseUsageQuery(raw string) (map[string]string,error) {
    values,err:=url.ParseQuery(raw);if err!=nil { return nil,err }
    result:=map[string]string{}
    for key,list:=range values { if (key!="start_date"&&key!="end_date")||len(list)!=1||list[0]=="" { return nil,errors.New("unsupported query") };result[key]=list[0] }
    return result,nil
}

func run() error {
    flags:=flag.NewFlagSet("aiw-agent-proxy",flag.ContinueOnError)
    path:=flags.String("config","","gateway JSON configuration")
    args:=os.Args[1:]
    action:="start"
    if len(args)>0&&(args[0]=="start"||args[0]=="stop") { action=args[0];args=args[1:] }
    if err:=flags.Parse(args);err!=nil { return err }
    if *path==""||flags.NArg()!=0 { return errors.New("usage: agent-gateway start|stop --config <file>") }
    if action=="stop" { return stopGateway(*path) }
    c,err:=loadConfig(*path);if err!=nil { return configError(err) }
	var s *Store
	if c.BackendMode=="codex" { s,err=openStore(c.StateDir);if err!=nil { return errors.New("cannot open gateway state: "+err.Error()) } }
    closeStore:=true
	defer func(){if closeStore&&s!=nil { s.Close() }}()
	if c.BackendMode=="codex" { if err=os.MkdirAll(c.WorkspaceDir,0700);err!=nil { return errors.New("cannot create workspace root") } }
    ctx,cancel:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM)
    defer cancel()
    gateway:=newGateway(ctx,c,s)
    gateway.shutdown=cancel
    server:=&http.Server{Addr:c.Listen,Handler:gateway,ReadHeaderTimeout:5*time.Second,ReadTimeout:10*time.Second,IdleTimeout:60*time.Second,MaxHeaderBytes:16<<10}
	if c.BackendMode=="openai_proxy" { server.ReadTimeout=0 }
    stopped:=make(chan error,1)
    cleanupDone:=make(chan struct{})
    go func(){
        <-ctx.Done()
        shutdown,done:=context.WithTimeout(context.Background(),10*time.Second);defer done()
		err:=server.Shutdown(shutdown)
		gateway.closeProxyConnections()
		if gateway.appPool!=nil { if cleanupErr:=gateway.appPool.close();cleanupErr!=nil&&err==nil { err=cleanupErr } }
        if err==nil {
            select { case <-cleanupDone:case <-shutdown.Done():err=shutdown.Err() }
        }
        stopped<-err
    }()
	go func(){
        defer close(cleanupDone)
        ticker:=time.NewTicker(time.Minute);defer ticker.Stop()
		for { select { case now:=<-ticker.C:if s!=nil { if err:=s.Clean(now.UTC());err!=nil { log.Print("content_cleanup_failed: new executions disabled") } };case <-ctx.Done():return } }
	}()
	log.Printf("gateway listening mode=%s backend_mode=%s address=%s",c.Mode,c.BackendMode,c.Listen)
    err=server.ListenAndServe()
    if !errors.Is(err,http.ErrServerClosed) { cancel();<-stopped;return errors.New("gateway listen failed") }
	if err=<-stopped;err!=nil { closeStore=false;_ = server.Close();return errors.New("gateway shutdown or backend cleanup failed; state lock preserved for operator reconciliation") }
    return nil
}

func main() {
    if err:=run();err!=nil { fmt.Fprintln(os.Stderr,err);os.Exit(1) }
}
