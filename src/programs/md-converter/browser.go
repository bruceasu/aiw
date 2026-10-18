package main

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strconv"
    "strings"
    "time"
)

func browserPath() (string,error) {
    if explicit:=os.Getenv("AIW_CONVERT_BROWSER");explicit!="" {p,e:=exec.LookPath(explicit);if e!=nil{return "",fmt.Errorf("browser-not-found: %w",e)};return p,nil}
    candidates:=[]string{"msedge","google-chrome","chromium","chromium-browser","microsoft-edge"}
    if runtime.GOOS=="windows" {
        for _,root:=range []string{os.Getenv("ProgramFiles"),os.Getenv("ProgramFiles(x86)"),os.Getenv("LocalAppData")} {
            if root!="" {candidates=append(candidates,filepath.Join(root,"Microsoft/Edge/Application/msedge.exe"),filepath.Join(root,"Google/Chrome/Application/chrome.exe"))}
        }
    } else if runtime.GOOS=="darwin" {
        candidates=append(candidates,"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome","/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge","/Applications/Chromium.app/Contents/MacOS/Chromium")
    }
    for _,candidate:=range candidates{if p,e:=exec.LookPath(candidate);e==nil{return p,nil}}
    return "",errors.New("browser-not-found: install Edge, Chrome or Chromium, or set AIW_CONVERT_BROWSER")
}

func printPDF(document string, destination string) error {
    browser,err:=browserPath();if err!=nil{return err}
    profile,err:=os.MkdirTemp("","aiw-pdf-profile-");if err!=nil{return err}
    defer os.RemoveAll(profile)
    ctx,cancel:=context.WithTimeout(context.Background(),90*time.Second)
    defer cancel()
    cmd:=exec.CommandContext(ctx,browser,"--headless=new","--disable-gpu","--no-first-run","--no-default-browser-check","--disable-extensions","--disable-sync","--disable-component-update","--disable-background-networking","--metrics-recording-only","--no-proxy-server","--remote-debugging-address=127.0.0.1","--remote-debugging-port=0","--user-data-dir="+profile,"about:blank")
    hideWindow(cmd)
    cmd.Stdout,cmd.Stderr=io.Discard,io.Discard
    if err=cmd.Start();err!=nil{return err}
    done:=make(chan error,1);go func(){done<-cmd.Wait()}()
    defer func(){cancel();_ = cmd.Process.Kill();select{case <-done:case <-time.After(3*time.Second):}}()
    var port string
    for {
        data,e:=os.ReadFile(filepath.Join(profile,"DevToolsActivePort"))
        if e==nil {port=strings.Split(string(data),"\n")[0];break}
        select{case <-ctx.Done():return errors.New("browser startup timed out");case e:=<-done:return fmt.Errorf("browser exited before CDP startup: %v",e);case <-time.After(50*time.Millisecond):}
    }
    number,err:=strconv.Atoi(strings.TrimSpace(port));if err!=nil||number<1||number>65535{return errors.New("invalid local CDP port")}
    // Only this loopback HTTP connection is used, never environment proxies.
    client:=&http.Client{Transport:&http.Transport{Proxy:nil},Timeout:5*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return errors.New("CDP redirect rejected")}}
    defer client.CloseIdleConnections()
    response,err:=client.Get("http://127.0.0.1:"+strconv.Itoa(number)+"/json/list");if err!=nil{return err}
    defer response.Body.Close()
    if response.StatusCode!=200{return errors.New("CDP target discovery failed")}
    var targets []struct{Type string `json:"type"`; URL string `json:"webSocketDebuggerUrl"`}
    if err=json.NewDecoder(io.LimitReader(response.Body,1024*1024)).Decode(&targets);err!=nil{return err}
    var address string
    for _,target:=range targets{if target.Type=="page"{address=target.URL;break}}
    deadline,_:=ctx.Deadline()
    c,err:=connectCDP(address,deadline);if err!=nil{return err};defer c.conn.Close()
    for _,method:=range []string{"Page.enable","Runtime.enable","Network.enable"}{if _,err=c.call(method,map[string]any{});err!=nil{return err}}
    if _,err=c.call("Network.setBlockedURLs",map[string]any{"urls":[]string{"http://*","https://*","file://*","ftp://*","ws://*","wss://*"}});err!=nil{return err}
    if _,err=c.call("Network.setBypassServiceWorker",map[string]any{"bypass":true});err!=nil{return err}
    tree,err:=c.call("Page.getFrameTree",map[string]any{});if err!=nil{return err}
    var frame struct{FrameTree struct{Frame struct{ID string `json:"id"`} `json:"frame"`} `json:"frameTree"`}
    if err=json.Unmarshal(tree,&frame);err!=nil{return err}
    if _,err=c.call("Page.setDocumentContent",map[string]any{"frameId":frame.FrameTree.Frame.ID,"html":document});err!=nil{return err}
    readiness:=`Promise.race([(async()=>{if(window.aiwMermaidReady){await window.aiwMermaidReady;}else if(document.querySelector('.mermaid')){throw Error('Mermaid is missing');}await document.fonts.ready;for(const f of document.fonts){if(f.status==='error')throw Error('Font failed: '+f.family);}await Promise.all(Array.from(document.images).map(async i=>{await i.decode();if(!i.naturalWidth)throw Error('Image failed');}));await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)));return true;})(),new Promise((_,r)=>setTimeout(()=>r(Error('Rendering timed out')),45000))])`
    evaluated,err:=c.call("Runtime.evaluate",map[string]any{"expression":readiness,"awaitPromise":true,"returnByValue":true});if err!=nil{return err}
    var ready struct{Exception json.RawMessage `json:"exceptionDetails"`;Result struct{Value bool `json:"value"`} `json:"result"`}
    if err=json.Unmarshal(evaluated,&ready);err!=nil{return err}
    if len(ready.Exception)>0||!ready.Result.Value{return fmt.Errorf("render readiness failed: %.1500s",evaluated)}
    printed,err:=c.call("Page.printToPDF",map[string]any{"printBackground":true,"preferCSSPageSize":true,"displayHeaderFooter":false});if err!=nil{return err}
    var pdf struct{Data string `json:"data"`};if err=json.Unmarshal(printed,&pdf);err!=nil{return err}
    data,err:=base64.StdEncoding.DecodeString(pdf.Data);if err!=nil{return err}
    if !strings.HasPrefix(string(data),"%PDF-"){return errors.New("browser returned invalid PDF")}
    return os.WriteFile(destination,data,0600)
}
