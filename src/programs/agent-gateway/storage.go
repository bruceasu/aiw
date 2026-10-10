package main

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "regexp"
    "sync"
    "time"
)

var recordName = regexp.MustCompile(`^resp_[0-9a-f]{32}\.json$`)

type Record struct {
    SchemaVersion int `json:"schema_version"`
    ID string `json:"id"`
    Principal string `json:"principal"`
    Model string `json:"model"`
    ReservedAt time.Time `json:"reserved_at"`
    StartedAt *time.Time `json:"started_at"`
    FinishedAt *time.Time `json:"finished_at"`
    State string `json:"state"`
    Usage *Usage `json:"usage"`
    ErrorCode string `json:"error_code,omitempty"`
}

type Content struct {
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt time.Time `json:"expires_at"`
    Input json.RawMessage `json:"input"`
    Instructions string `json:"instructions,omitempty"`
    Output []string `json:"output"`
    State string `json:"state"`
}

type Store struct {
    mu sync.Mutex
    dir string
    records map[string]Record
    requests map[string]HTTPRequestRecord
    fatal bool
    lock *os.File
    uncertain map[string]bool
}

func atomicJSON(path string, value any) error {
    data,err:=json.Marshal(value);if err!=nil { return err }
    file,err:=os.CreateTemp(filepath.Dir(path),".pending-");if err!=nil { return err }
    name:=file.Name()
    defer os.Remove(name)
    if err=file.Chmod(0600);err==nil { _,err=file.Write(data) }
    if err==nil { err=file.Sync() }
    closeErr:=file.Close();if err==nil { err=closeErr }
    if err!=nil { return err }
    return os.Rename(name,path)
}

func openStore(dir string) (*Store,error) {
    if err:=os.MkdirAll(dir,0700);err!=nil { return nil,errors.New("cannot create state directory") }
    lock,err:=os.OpenFile(filepath.Join(dir,"gateway.lock"),os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600)
    if err!=nil { return nil,errors.New("state directory locked; operator must reconcile stale lock") }
    s:=&Store{dir:dir,records:map[string]Record{},requests:map[string]HTTPRequestRecord{},lock:lock,uncertain:map[string]bool{}}
    good:=false
    defer func(){if !good { s.Close() }}()
    manifest:=filepath.Join(dir,"manifest.json")
    data,err:=os.ReadFile(manifest)
    initializing:=os.IsNotExist(err)
    if os.IsNotExist(err) {
        entries,e:=os.ReadDir(dir);if e!=nil { return nil,e }
        for _,entry:=range entries { if entry.Name()!="gateway.lock" { return nil,errors.New("nonempty state without manifest") } }
        if err=atomicJSON(manifest,map[string]int{"schema_version":1});err!=nil { return nil,err }
    } else {
        var version struct{Version int `json:"schema_version"`}
        if err!=nil||strictDecode(data,&version)!=nil||version.Version!=1 { return nil,errors.New("invalid state manifest") }
    }
    for _,name:=range []string{"metadata","content"} {
        path:=filepath.Join(dir,name)
        if initializing { if err=os.MkdirAll(path,0700);err!=nil { return nil,err }
        } else { info,e:=os.Lstat(path);if e!=nil||!info.IsDir()||info.Mode()&os.ModeSymlink!=0 { return nil,errors.New("existing state directory is incomplete") } }
    }
    entries,err:=os.ReadDir(filepath.Join(dir,"metadata"));if err!=nil { return nil,err }
    for _,entry:=range entries {
        if entry.Type()&os.ModeSymlink!=0||entry.IsDir() { return nil,errors.New("invalid metadata entry") }
        if !recordName.MatchString(entry.Name()) {
            if len(entry.Name())>=9&&entry.Name()[:9]==".pending-" { if err=os.Remove(filepath.Join(dir,"metadata",entry.Name()));err!=nil { return nil,err };continue }
            return nil,errors.New("unexpected metadata filename")
        }
        data,e:=os.ReadFile(filepath.Join(dir,"metadata",entry.Name()));if e!=nil||len(data)>maxBody { return nil,errors.New("cannot read metadata") }
        var r Record
        if strictDecode(data,&r)!=nil||r.SchemaVersion!=1||r.ID+".json"!=entry.Name()||r.Principal==""||r.Model==""||r.ReservedAt.IsZero() { return nil,errors.New("corrupt request metadata") }
        switch r.State {
        case "reserved","spawn_failed":
            if r.StartedAt!=nil { return nil,errors.New("invalid unstarted metadata") }
        case "started","succeeded","failed","timed_out","cancelled":
            if r.StartedAt==nil||r.StartedAt.IsZero() { return nil,errors.New("missing start time") }
        default:return nil,errors.New("invalid request state")
        }
        if r.Usage!=nil&&(r.Usage.InputTokens<0||r.Usage.OutputTokens<0||r.Usage.TotalTokens!=r.Usage.InputTokens+r.Usage.OutputTokens) { return nil,errors.New("invalid token metadata") }
        if r.State=="started" {
            now:=time.Now().UTC();r.State="failed";r.FinishedAt=&now;r.ErrorCode="backend_interrupted"
            if e=atomicJSON(filepath.Join(dir,"metadata",entry.Name()),r);e!=nil { return nil,e }
        }
        s.records[r.ID]=r
        if r.State=="reserved" { s.uncertain[r.Principal]=true }
    }
    if err=s.loadHTTPRequests();err!=nil { return nil,err }
    if err=s.cleanLocked(time.Now().UTC());err!=nil { return nil,err }
    good=true
    return s,nil
}

func (s *Store) Close() {
    if s.lock!=nil { _=s.lock.Close();_ = os.Remove(filepath.Join(s.dir,"gateway.lock"));s.lock=nil }
}

func recordDate(r Record,zone *time.Location) string {
    at:=r.ReservedAt;if r.StartedAt!=nil { at=*r.StartedAt };return at.In(zone).Format("2006-01-02")
}

func (s *Store) countLocked(principal,date string,zone *time.Location) int {
    count:=0
    // A reservation holds capacity across midnight until its actual start is
    // committed under this same mutex. New-day admission cannot outrun it.
    for _,r:=range s.records {
        if r.Principal!=principal||r.State=="spawn_failed" { continue }
        if r.State=="reserved"||(r.StartedAt!=nil&&recordDate(r,zone)==date) { count++ }
    }
    return count
}

func (s *Store) Reserve(r Record,p Principal,zone *time.Location) error {
    s.mu.Lock();defer s.mu.Unlock()
    if s.fatal { return errors.New("storage_error") }
    if s.uncertain[p.ID] { return errors.New("unresolved_reservation") }
    // Admission may wait for this mutex across midnight. ReservedAt is only
    // the audit/content timestamp; quota uses the day observed while locked.
    today:=time.Now().In(zone).Format("2006-01-02")
    if s.countLocked(p.ID,today,zone)>=p.DailyLimit { return errors.New("daily_limit") }
    if err:=s.putLocked(r);err!=nil { return errors.New("storage_error") };return nil
}

func (s *Store) putLocked(r Record) error {
    if !recordName.MatchString(r.ID+".json") { return errors.New("invalid request id") }
    if err:=atomicJSON(filepath.Join(s.dir,"metadata",r.ID+".json"),r);err!=nil { s.fatal=true;return err }
    s.records[r.ID]=r;return nil
}

func (s *Store) Put(r Record) error { s.mu.Lock();defer s.mu.Unlock();return s.putLocked(r) }

func (s *Store) SaveContent(id string,c Content) error {
    s.mu.Lock();defer s.mu.Unlock()
    if !recordName.MatchString(id+".json") { return errors.New("invalid request id") }
    path:=filepath.Join(s.dir,"content",id+".json")
    if !time.Now().UTC().Before(c.ExpiresAt) {
        err:=os.Remove(path);if os.IsNotExist(err) { return nil };if err!=nil { s.fatal=true };return err
    }
    err:=atomicJSON(path,c);if err!=nil { s.fatal=true };return err
}

func (s *Store) cleanLocked(now time.Time) error {
    entries,err:=os.ReadDir(filepath.Join(s.dir,"content"));if err!=nil { return err }
    for _,entry:=range entries {
        if entry.Type()&os.ModeSymlink!=0||entry.IsDir() { return errors.New("invalid content entry") }
        if !recordName.MatchString(entry.Name()) {
            if len(entry.Name())>=9&&entry.Name()[:9]==".pending-" { if err=os.Remove(filepath.Join(s.dir,"content",entry.Name()));err!=nil { return err };continue };return errors.New("unexpected content filename")
        }
        path:=filepath.Join(s.dir,"content",entry.Name())
        data,e:=os.ReadFile(path);if e!=nil||len(data)>maxWire { return errors.New("cannot read content record") }
        var c Content
        if strictDecode(data,&c)!=nil||c.CreatedAt.IsZero()||!c.ExpiresAt.Equal(c.CreatedAt.Add(7*24*time.Hour)) { return errors.New("invalid content expiry") }
        if !now.Before(c.ExpiresAt) { if err=os.Remove(path);err!=nil { return err } }
    }
    return nil
}

func (s *Store) Clean(now time.Time) error { s.mu.Lock();defer s.mu.Unlock();err:=s.cleanLocked(now);if err!=nil { s.fatal=true };return err }

func (s *Store) Stop() { s.mu.Lock();s.fatal=true;s.mu.Unlock() }

type Group struct {
    Date string `json:"date"`
    Model string `json:"model"`
    Requests int `json:"requests"`
    Succeeded int `json:"succeeded"`
    Failed int `json:"failed"`
    TimedOut int `json:"timed_out"`
    Cancelled int `json:"cancelled"`
    InProgress int `json:"in_progress"`
    InputTokens int64 `json:"known_input_tokens"`
    OutputTokens int64 `json:"known_output_tokens"`
    TotalTokens int64 `json:"known_total_tokens"`
    UnknownUsageRequests int `json:"unknown_usage_requests"`
    Cost any `json:"cost"`
}

func (s *Store) Aggregate(p Principal,start,end string,zone *time.Location) ([]Group,int,int) {
    s.mu.Lock();defer s.mu.Unlock()
    groups:=map[string]*Group{}
    for _,r:=range s.records {
        date:=recordDate(r,zone)
        if r.Principal!=p.ID||r.StartedAt==nil||date<start||date>end { continue }
        key:=date+"\x00"+r.Model
        g:=groups[key];if g==nil { g=&Group{Date:date,Model:r.Model};groups[key]=g }
        g.Requests++
        switch r.State { case "succeeded":g.Succeeded++;case "failed":g.Failed++;case "timed_out":g.TimedOut++;case "cancelled":g.Cancelled++;default:g.InProgress++ }
        if r.Usage==nil { g.UnknownUsageRequests++ } else { g.InputTokens+=r.Usage.InputTokens;g.OutputTokens+=r.Usage.OutputTokens;g.TotalTokens+=r.Usage.TotalTokens }
    }
    result:=[]Group{}
    for _,g:=range groups { result=append(result,*g) }
    used,reserved:=0,0
    today:=time.Now().In(zone).Format("2006-01-02")
    for _,r:=range s.records {
        if r.Principal!=p.ID||r.State=="spawn_failed" { continue }
        if r.State=="reserved" { reserved++ } else if r.StartedAt!=nil&&recordDate(r,zone)==today { used++ }
    }
    return result,used,reserved
}
