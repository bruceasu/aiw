package main

import (
    "crypto/subtle"
    "encoding/json"
    "errors"
    "fmt"
    "net"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
    _ "time/tzdata"
)

const maxBody = 1 << 20
const maxText = 64 << 10
const maxOutput = 256 << 10
const maxWire = 4 << 20

type Principal struct {
    ID string `json:"id"`
    Enabled bool `json:"enabled"`
    Keys []string `json:"keys,omitempty"`
    AllowedModels []string `json:"allowed_models"`
    RPM int `json:"rpm"`
    DailyLimit int `json:"daily_limit"`
    Concurrency int `json:"concurrency"`
}

type Config struct {
    Mode string `json:"mode"`
    Listen string `json:"listen"`
    StateDir string `json:"state_dir"`
    WorkspaceDir string `json:"workspace_dir"`
    CodexPath string `json:"codex_path"`
    CodexScript string `json:"codex_script,omitempty"`
    Timezone string `json:"timezone"`
    TimeoutSeconds int `json:"timeout_seconds"`
    GlobalConcurrency int `json:"global_concurrency"`
    Models map[string]string `json:"models"`
    Principals []Principal `json:"principals"`
    zone *time.Location
}

func loadConfig(path string) (Config, error) {
    c := Config{Mode:"development", Listen:"127.0.0.1:43127", Timezone:"UTC", TimeoutSeconds:600, GlobalConcurrency:4}
    data, err := os.ReadFile(path)
    if err != nil { return c, errors.New("cannot read gateway config") }
    if len(data)>maxBody { return c, errors.New("config exceeds size limit") }
    if err = strictDecode(data, &c); err != nil { return c, errors.New("invalid gateway config schema") }
    if c.Mode!="development" && c.Mode!="shared" { return c, errors.New("invalid mode") }
    var raw map[string]json.RawMessage
    _ = json.Unmarshal(data,&raw)
    if c.Mode=="shared" && raw["listen"]==nil { return c, errors.New("shared mode requires explicit listen") }
    host,port,err := net.SplitHostPort(c.Listen)
    if err!=nil { return c, errors.New("listen must be an IP address and port") }
    number,err:=strconv.Atoi(port)
    if err!=nil||number<1||number>65535 { return c,errors.New("invalid listen port") }
    ip:=net.ParseIP(host)
    if ip==nil || (c.Mode=="development" && !ip.IsLoopback()) { return c, errors.New("invalid listen address for mode") }
    if c.TimeoutSeconds<1 || c.TimeoutSeconds>3600 || c.GlobalConcurrency<1 || c.GlobalConcurrency>256 { return c, errors.New("invalid execution limits") }
    c.zone,err=time.LoadLocation(c.Timezone)
    if err!=nil { return c, errors.New("invalid timezone") }
    for _,p:=range []string{c.StateDir,c.WorkspaceDir,c.CodexPath} {
        if !filepath.IsAbs(p) { return c, errors.New("server paths must be absolute") }
    }
    a,b:=filepath.Clean(c.StateDir),filepath.Clean(c.WorkspaceDir)
    inside:=func(x,y string) bool { rel,e:=filepath.Rel(x,y); return e==nil && (rel=="." || (!strings.HasPrefix(rel,".."+string(os.PathSeparator)) && rel!="..")) }
    if inside(a,b)||inside(b,a) { return c, errors.New("state and workspace directories must be separate") }
    if info,e:=os.Stat(c.CodexPath); e!=nil||!info.Mode().IsRegular() { return c, errors.New("codex executable is missing") }
    if c.CodexScript!="" {
        info,e:=os.Stat(c.CodexScript)
        if !filepath.IsAbs(c.CodexScript)||e!=nil||!info.Mode().IsRegular() { return c, errors.New("codex script is missing") }
    }
    if err=validateExecutable(c); err!=nil { return c,err }
    if len(c.Models)==0||len(c.Principals)==0 { return c, errors.New("models and principals are required") }
    for id,backend:=range c.Models { if strings.TrimSpace(id)==""||strings.TrimSpace(backend)==""||strings.ContainsAny(id,"\r\n") { return c,errors.New("invalid model mapping") } }
    ids,keys:=map[string]bool{},map[string]bool{}
    enabled:=false
    for _,p:=range c.Principals {
        if p.ID==""||ids[p.ID]||p.RPM<1||p.RPM>100000||p.DailyLimit<1||p.DailyLimit>10000000||p.Concurrency<1||p.Concurrency>c.GlobalConcurrency { return c,errors.New("invalid principal or limits") }
        ids[p.ID]=true
        if len(p.Keys)==0||len(p.AllowedModels)==0 { return c,errors.New("principal requires keys and models") }
        for _,key:=range p.Keys {
            if !validKey(key) { return c,errors.New("invalid gateway key") }
            if keys[key] { return c,errors.New("duplicate gateway credential") }
            keys[key]=true
        }
        for _,model:=range p.AllowedModels { if _,ok:=c.Models[model];!ok { return c,errors.New("unknown principal model") } }
        enabled=enabled||p.Enabled
    }
    if !enabled { return c,errors.New("at least one enabled principal is required") }
    return c,nil
}

func (c Config) authenticate(header string) *Principal {
    if !strings.HasPrefix(header,"Bearer ") { return nil }
    key:=strings.TrimPrefix(header,"Bearer ")
    if !validKey(key) { return nil }
    var found *Principal
    for i:=range c.Principals {
        p:=&c.Principals[i]
        for _,configured:=range p.Keys {
            if subtle.ConstantTimeCompare([]byte(key),[]byte(configured))==1 {
                if p.Enabled { found=p } else { return nil }
            }
        }
    }
    return found
}

func validKey(key string) bool { return len(key)>=43&&len(key)<=1024&&!strings.ContainsAny(key," \r\n\t") }

func (p Principal) allows(model string) bool {
    for _,m:=range p.AllowedModels { if m==model { return true } }; return false
}

func childEnv() []string {
    allowed:=map[string]bool{}
    for _,key:=range []string{"PATH","HOME","USERPROFILE","SYSTEMROOT","WINDIR","APPDATA","LOCALAPPDATA","TEMP","TMP","CODEX_HOME","HTTPS_PROXY","HTTP_PROXY","NO_PROXY"} { allowed[key]=true }
    result:=[]string{}
    for _,entry:=range os.Environ() { key,_,ok:=strings.Cut(entry,"="); if ok&&allowed[strings.ToUpper(key)] { result=append(result,entry) } }
    return result
}

func configError(err error) error { return fmt.Errorf("gateway configuration: %w",err) }
