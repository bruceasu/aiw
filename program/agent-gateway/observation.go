package main

import (
    "encoding/json"
    "errors"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

// HTTP records are separate from execution metadata: rejection never consumes
// a reservation, daily quota, or tokens.
type HTTPRequestRecord struct {
    SchemaVersion int `json:"schema_version"`
    ID string `json:"request_id"`
    Principal string `json:"principal"`
    Method string `json:"method"`
    Route string `json:"route"`
    Model string `json:"model"`
    ReceivedAt time.Time `json:"received_at"`
    FinishedAt *time.Time `json:"finished_at"`
    DurationMS *int64 `json:"duration_ms"`
    HTTPStatus int `json:"http_status"`
    State string `json:"state"`
    ErrorCode string `json:"error_code,omitempty"`
    ExecutionStarted bool `json:"execution_started"`
    DeliveryFailed bool `json:"delivery_failed"`
}

func observedRoute(path string) string {
    switch path { case "/v1/responses","/v1/chat/completions","/v1/models","/v1/usage":return path }
    return "other"
}

func observedMethod(method string) string {
    switch method { case "GET","POST","PUT","PATCH","DELETE","HEAD","OPTIONS","CONNECT","TRACE":return method }
    return "other"
}

func (s *Store) loadHTTPRequests() error {
    manifest:=filepath.Join(s.dir,"requests-manifest.json")
    manifestData,manifestErr:=os.ReadFile(manifest)
    upgrading:=os.IsNotExist(manifestErr)
    if !upgrading {
        var version struct { Version int `json:"schema_version"` }
        if manifestErr!=nil||len(manifestData)>maxBody||strictDecode(manifestData,&version)!=nil||version.Version!=1 {
            return errors.New("invalid HTTP request manifest")
        }
    }
    dir:=filepath.Join(s.dir,"requests")
    info,err:=os.Lstat(dir)
    if os.IsNotExist(err) {
        if !upgrading { return errors.New("HTTP request directory is missing") }
        // Additive upgrade for existing execution-only stores.
        if err=os.Mkdir(dir,0700);err!=nil { return err }
    } else if err!=nil||!info.IsDir()||info.Mode()&os.ModeSymlink!=0 {
        return errors.New("invalid HTTP request directory")
    }
    entries,err:=os.ReadDir(dir);if err!=nil { return err }
    for _,entry:=range entries {
        if !entry.Type().IsRegular() { return errors.New("invalid HTTP request entry") }
        if !recordName.MatchString(entry.Name()) {
            if strings.HasPrefix(entry.Name(),".pending-") {
                if err=os.Remove(filepath.Join(dir,entry.Name()));err!=nil { return err };continue
            }
            return errors.New("unexpected HTTP request filename")
        }
        data,e:=os.ReadFile(filepath.Join(dir,entry.Name()))
        if e!=nil||len(data)>maxBody { return errors.New("cannot read HTTP request record") }
        var r HTTPRequestRecord
        if strictDecode(data,&r)!=nil||r.SchemaVersion!=1||r.ID+".json"!=entry.Name()||r.ReceivedAt.IsZero()||observedRoute(r.Route)!=r.Route||observedMethod(r.Method)!=r.Method {
            return errors.New("corrupt HTTP request record")
        }
        if r.HTTPStatus!=0&&(r.HTTPStatus<200||r.HTTPStatus>599) { return errors.New("invalid HTTP request status") }
        if r.DurationMS!=nil&&*r.DurationMS<0 { return errors.New("invalid HTTP request duration") }
        switch r.State {
        case "in_progress":
            if r.FinishedAt!=nil||r.DurationMS!=nil||r.HTTPStatus!=0 { return errors.New("invalid pending HTTP request") }
            r.State="interrupted";r.ErrorCode="request_interrupted"
            if execution,ok:=s.records[r.ID];ok { r.ExecutionStarted=execution.StartedAt!=nil }
            // The actual end time and elapsed duration are unknown after a crash.
            if e=atomicJSON(filepath.Join(dir,entry.Name()),r);e!=nil { return e }
        case "interrupted":
            if r.FinishedAt!=nil||r.DurationMS!=nil { return errors.New("invalid interrupted HTTP request") }
        case "succeeded","rejected","failed","timed_out","cancelled":
            if r.FinishedAt==nil||r.FinishedAt.IsZero()||r.DurationMS==nil { return errors.New("invalid completed HTTP request") }
        default:return errors.New("invalid HTTP request state")
        }
        s.requests[r.ID]=r
    }
    if upgrading { return atomicJSON(manifest,map[string]int{"schema_version":1}) }
    return nil
}

func (s *Store) PutHTTPRequest(r HTTPRequestRecord) error {
    s.mu.Lock();defer s.mu.Unlock()
    if !recordName.MatchString(r.ID+".json") { return errors.New("invalid HTTP request id") }
    if s.fatal { return errors.New("storage_error") }
    if err:=atomicJSON(filepath.Join(s.dir,"requests",r.ID+".json"),r);err!=nil { s.fatal=true;return errors.New("storage_error") }
    s.requests[r.ID]=r
    return nil
}

type HTTPGroup struct {
    Date string `json:"date"`
    Model string `json:"model"`
    Route string `json:"route"`
    Method string `json:"method"`
    Requests int `json:"requests"`
    Succeeded int `json:"succeeded"`
    Rejected int `json:"rejected"`
    Failed int `json:"failed"`
    TimedOut int `json:"timed_out"`
    Cancelled int `json:"cancelled"`
    Interrupted int `json:"interrupted"`
    InProgress int `json:"in_progress"`
    Executed int `json:"executed"`
    DeliveryFailed int `json:"delivery_failed"`
    DurationSamples int `json:"duration_samples"`
    TotalDurationMS int64 `json:"total_duration_ms"`
    HTTPStatuses map[int]int `json:"http_statuses"`
    ErrorCodes map[string]int `json:"error_codes"`
}

func (s *Store) AggregateHTTP(p Principal,start,end string,zone *time.Location) []HTTPGroup {
    s.mu.Lock();defer s.mu.Unlock()
    type groupKey struct { date,model,route,method string }
    groups:=map[groupKey]*HTTPGroup{}
    for _,r:=range s.requests {
        date:=r.ReceivedAt.In(zone).Format("2006-01-02")
        if r.Principal!=p.ID||date<start||date>end { continue }
        model:=r.Model;executed:=r.ExecutionStarted
        if execution,ok:=s.records[r.ID];ok { model=execution.Model;executed=execution.StartedAt!=nil }
        key:=groupKey{date,model,r.Route,r.Method}
        g:=groups[key]
        if g==nil { g=&HTTPGroup{Date:date,Model:model,Route:r.Route,Method:r.Method,HTTPStatuses:map[int]int{},ErrorCodes:map[string]int{}};groups[key]=g }
        g.Requests++
        switch r.State {
        case "succeeded":g.Succeeded++
        case "rejected":g.Rejected++
        case "failed":g.Failed++
        case "timed_out":g.TimedOut++
        case "cancelled":g.Cancelled++
        case "interrupted":g.Interrupted++
        default:g.InProgress++
        }
        if executed { g.Executed++ }
        if r.DeliveryFailed { g.DeliveryFailed++ }
        if r.HTTPStatus!=0 { g.HTTPStatuses[r.HTTPStatus]++ }
        if r.ErrorCode!="" { g.ErrorCodes[r.ErrorCode]++ }
        if r.DurationMS!=nil { g.DurationSamples++;g.TotalDurationMS+=*r.DurationMS }
    }
    result:=[]HTTPGroup{}
    for _,g:=range groups { result=append(result,*g) }
    sort.Slice(result,func(i,j int)bool {
        a,b:=result[i],result[j]
        if a.Date!=b.Date { return a.Date<b.Date };if a.Model!=b.Model { return a.Model<b.Model }
        if a.Route!=b.Route { return a.Route<b.Route };return a.Method<b.Method
    })
    return result
}

// Unwrap preserves ResponseController deadlines and flushing for SSE.
type observedWriter struct {
    http.ResponseWriter
    record HTTPRequestRecord
    status int
    deliveryFailed bool
}

func (w *observedWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *observedWriter) WriteHeader(status int) {
    if w.status==0&&status>=200 { w.status=status }
    w.ResponseWriter.WriteHeader(status)
}

func (w *observedWriter) Write(data []byte) (int,error) {
    if w.status==0 { w.WriteHeader(http.StatusOK) }
    n,err:=w.ResponseWriter.Write(data)
    if err!=nil { w.deliveryFailed=true }
    return n,err
}

func (w *observedWriter) FlushError() error {
    if w.status==0 { w.WriteHeader(http.StatusOK) }
    err:=http.NewResponseController(w.ResponseWriter).Flush()
    if err!=nil { w.deliveryFailed=true };return err
}

func logHTTPRequest(event string,r HTTPRequestRecord) {
    data,err:=json.Marshal(struct {
        Event string `json:"event"`
        HTTPRequestRecord
    }{Event:event,HTTPRequestRecord:r})
    if err!=nil { log.Print("request_log_encoding_failed");return }
    log.Print(string(data))
}

func (g *Gateway) finishHTTPRequest(w *observedWriter,r *http.Request,started time.Time,panicValue any) {
    now:=time.Now().UTC();duration:=time.Since(started).Milliseconds()
    w.record.FinishedAt=&now;w.record.DurationMS=&duration;w.record.HTTPStatus=w.status
    w.record.DeliveryFailed=w.deliveryFailed
    if panicValue!=nil {
        w.record.State="failed";w.record.ErrorCode="handler_panic"
    } else if w.record.State=="in_progress" {
        w.record.State="succeeded"
        if w.status>=400 { w.record.State="rejected" }
    }
    if (w.deliveryFailed||r.Context().Err()!=nil)&&w.record.State=="succeeded" {
        w.record.State="failed";w.record.ErrorCode="delivery_error"
        if r.Context().Err()!=nil { w.record.State="cancelled";w.record.ErrorCode="cancelled" }
    }
    if err:=g.store.PutHTTPRequest(w.record);err!=nil {
        log.Printf("request_audit_failed request_id=%s principal=%q",w.record.ID,w.record.Principal)
    }
    logHTTPRequest("request_finished",w.record)
}
