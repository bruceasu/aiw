package main

import (
    "context"
    "bufio"
    "encoding/json"
    "errors"
    "io"
    "net"
    "net/http"
    "net/http/httputil"
    "sort"
    "strconv"
    "strings"
    "sync"
    "time"
)

func intString(value int) string { return strconv.Itoa(value) }

type Gateway struct {
    config Config
    store *Store
    appPool *appPool
	proxy http.Handler
    ctx context.Context
    mu sync.Mutex
    active int
    principalActive map[string]int
    rpm map[string][]time.Time
    shutdown context.CancelFunc
    proxyConns map[net.Conn]struct{}
}

type proxyRequestIdentity struct {
	principal Principal
	requestID string
}

type proxyIdentityContextKey struct{}

func newGateway(ctx context.Context,c Config,s *Store) *Gateway {
	var pool *appPool
	if c.BackendMode=="codex" { pool=newAppPool(c) }
	g:=&Gateway{config:c,store:s,ctx:ctx,appPool:pool,principalActive:map[string]int{},rpm:map[string][]time.Time{},proxyConns:map[net.Conn]struct{}{}}
	if c.BackendMode=="openai_proxy" { g.proxy=g.newProxyHandler() }
	return g
}

func (g *Gateway) newProxyHandler() http.Handler {
	target:=*g.config.proxyURL
	return &httputil.ReverseProxy{Rewrite:func(request *httputil.ProxyRequest) {
		identity,_:=request.In.Context().Value(proxyIdentityContextKey{}).(proxyRequestIdentity)
		request.SetURL(&target)
		suffix:=strings.TrimPrefix(request.In.URL.Path,"/v1")
		request.Out.URL.Path=strings.TrimSuffix(target.Path,"/")+suffix
		if request.In.URL.RawPath!="" {
			escapedSuffix:=strings.TrimPrefix(request.In.URL.EscapedPath(),"/v1")
			request.Out.URL.RawPath=strings.TrimSuffix(target.EscapedPath(),"/")+escapedSuffix
		}
		request.Out.Host=target.Host
		request.Out.Header.Del("Authorization")
		request.Out.Header.Del("Api-Key")
		request.Out.Header.Del("X-Api-Key")
		request.Out.Header.Del("Openai-Organization")
		request.Out.Header.Del("Openai-Project")
		request.Out.Header.Del("X-Openai-Client-User")
		request.Out.Header.Set("Authorization","Bearer "+identity.principal.upstreamAPIKey)
		request.Out.Header.Set("X-Request-Id",identity.requestID)
	},ErrorHandler:func(w http.ResponseWriter,r *http.Request,err error) {
		if r.Context().Err()!=nil { return }
		fail(w,http.StatusBadGateway,"upstream_unavailable")
	}}
}

type proxyTrackedWriter struct {
	http.ResponseWriter
	onHijack func(net.Conn)
	onClose func(net.Conn)
}

func (w *proxyTrackedWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *proxyTrackedWriter) Hijack() (net.Conn,*bufio.ReadWriter,error) {
	hijacker,ok:=w.ResponseWriter.(http.Hijacker)
	if !ok { return nil,nil,errors.New("hijacking is unsupported") }
	conn,rw,err:=hijacker.Hijack()
	if err!=nil { return nil,nil,err }
	tracked:=&trackedProxyConn{Conn:conn,onClose:w.onClose}
	w.onHijack(tracked)
	return tracked,rw,nil
}

type trackedProxyConn struct { net.Conn;onClose func(net.Conn);once sync.Once }

func (c *trackedProxyConn) Close() error {
	err:=c.Conn.Close()
	c.once.Do(func(){ if c.onClose!=nil { c.onClose(c) } })
	return err
}

func (g *Gateway) trackProxyConnection(conn net.Conn) { g.mu.Lock();g.proxyConns[conn]=struct{}{};g.mu.Unlock() }
func (g *Gateway) untrackProxyConnection(conn net.Conn) { g.mu.Lock();delete(g.proxyConns,conn);g.mu.Unlock() }
func (g *Gateway) closeProxyConnections() {
	g.mu.Lock();connections:=make([]net.Conn,0,len(g.proxyConns));for conn:=range g.proxyConns { connections=append(connections,conn) };g.mu.Unlock()
	for _,conn:=range connections { _=conn.Close() }
}

func writeJSON(w http.ResponseWriter,status int,data any) {
    _=http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5*time.Second))
    w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(data)
}

func fail(w http.ResponseWriter,status int,code string) {
    if observed,ok:=w.(*observedWriter);ok {
        observed.record.ErrorCode=code
        if observed.record.State=="in_progress" { observed.record.State="rejected" }
    }
    kind:="invalid_request_error"
    switch status { case 401:kind="authentication_error";case 429:kind="rate_limit_error";case 500,502,503,504:kind="server_error" }
    if status==429||status==503 { w.Header().Set("Retry-After","60") }
    writeJSON(w,status,map[string]any{"error":map[string]any{"message":"Gateway request rejected: "+code,"type":kind,"param":nil,"code":code}})
}

func (g *Gateway) acquire(p Principal) string {
    g.mu.Lock();defer g.mu.Unlock()
    now:=time.Now();old:=g.rpm[p.ID];times:=old[:0]
    for _,at:=range old { if now.Sub(at)<time.Minute { times=append(times,at) } }
    g.rpm[p.ID]=times
    if len(times)>=p.RPM { return "rpm_limit" }
    g.rpm[p.ID]=append(times,now)
    if g.active>=g.config.GlobalConcurrency||g.principalActive[p.ID]>=p.Concurrency { return "capacity_error" }
    g.active++;g.principalActive[p.ID]++;return ""
}

func (g *Gateway) release(p Principal) { g.mu.Lock();g.active--;g.principalActive[p.ID]--;g.mu.Unlock() }

func (g *Gateway) ServeHTTP(w http.ResponseWriter,r *http.Request) {
    started:=time.Now()
    id:=newID();w.Header().Set("x-request-id",id)
    observed:=&observedWriter{ResponseWriter:w,record:HTTPRequestRecord{SchemaVersion:1,ID:id,Method:observedMethod(r.Method),Route:observedRoute(r.URL.Path),ReceivedAt:started.UTC(),State:"in_progress"}}
    w=observed
    rejection:=""
    if g.config.Mode=="development" {
        host,_,err:=net.SplitHostPort(r.RemoteAddr);ip:=net.ParseIP(host)
        if err!=nil||ip==nil||!ip.IsLoopback() { rejection="local_only" }
    }
    var p *Principal
    if rejection=="" {
        p=g.config.authenticate(r.Header.Get("Authorization"))
        if p==nil { rejection="invalid_api_key" } else { observed.record.Principal=p.ID }
    }
    logHTTPRequest("request_started",observed.record)
    defer func(){
        panicValue:=recover()
        g.finishHTTPRequest(observed,r,started,panicValue)
        if panicValue!=nil { panic(panicValue) }
    }()
	if g.store!=nil { if err:=g.store.PutHTTPRequest(observed.record);err!=nil {
        if r.URL.Path!="/internal/shutdown" { fail(w,503,"storage_error");return }
        logShutdownAuditFailure()
	} }
    if rejection!="" { fail(w,401,rejection);return }
    select { case <-g.ctx.Done():fail(w,503,"shutting_down");return;default: }
	if r.URL.Path=="/internal/shutdown" { g.requestShutdown(w,r);return }
	if g.config.BackendMode=="openai_proxy" {
		if r.URL.Path!="/v1"&&!strings.HasPrefix(r.URL.Path,"/v1/") { fail(w,404,"not_found");return }
		if code:=g.acquire(*p);code!="" { status:=429;if code=="capacity_error" { status=503 };fail(w,status,code);return }
		defer g.release(*p)
		if g.proxy==nil { fail(w,502,"upstream_unavailable");return }
		requestCtx,cancel:=context.WithCancel(r.Context())
		stop:=context.AfterFunc(g.ctx,cancel)
		defer func(){stop();cancel()}()
		ctx:=context.WithValue(requestCtx,proxyIdentityContextKey{},proxyRequestIdentity{principal:*p,requestID:id})
		trackedWriter:=&proxyTrackedWriter{ResponseWriter:w,onHijack:g.trackProxyConnection,onClose:g.untrackProxyConnection}
		g.proxy.ServeHTTP(trackedWriter,r.WithContext(ctx))
		return
	}
    switch r.URL.Path {
    case "/v1/models":
        if r.Method!="GET" { fail(w,405,"method_not_allowed");return }
        if r.URL.RawQuery!="" { fail(w,400,"unsupported_query");return }
        models:=[]any{}
        for _,model:=range p.AllowedModels { models=append(models,map[string]any{"id":model,"object":"model","created":0,"owned_by":"aiw"}) }
        writeJSON(w,200,map[string]any{"object":"list","data":models})
    case "/v1/usage":g.usage(w,r,*p)
    case "/v1/responses":g.responses(w,r,*p,id)
    case "/v1/chat/completions":g.chatCompletions(w,r,*p,id)
    default:fail(w,404,"not_found")
    }
}

func (g *Gateway) usage(w http.ResponseWriter,r *http.Request,p Principal) {
    if r.Method!="GET" { fail(w,405,"method_not_allowed");return }
    query,err:=parseUsageQuery(r.URL.RawQuery)
    if err!=nil { fail(w,400,"invalid_date_range");return }
    today:=time.Now().In(g.config.zone).Format("2006-01-02")
    start,end:=query["start_date"],query["end_date"]
    if start=="" { start=today };if end=="" { end=today }
    first,e1:=time.ParseInLocation("2006-01-02",start,g.config.zone);last,e2:=time.ParseInLocation("2006-01-02",end,g.config.zone)
    if e1!=nil||e2!=nil||first.After(last)||first.Format("2006-01-02")!=start||last.Format("2006-01-02")!=end||first.AddDate(0,0,365).Before(last) { fail(w,400,"invalid_date_range");return }
    groups,used,reserved:=g.store.Aggregate(p,start,end,g.config.zone)
    sort.Slice(groups,func(i,j int)bool{if groups[i].Date==groups[j].Date{return groups[i].Model<groups[j].Model};return groups[i].Date<groups[j].Date})
    httpGroups:=g.store.AggregateHTTP(p,start,end,g.config.zone)
    writeJSON(w,200,map[string]any{"principal":p.ID,"timezone":g.config.Timezone,"start_date":start,"end_date":end,"daily_limit":p.DailyLimit,"today_used":used,"today_reserved":reserved,"groups":groups,"http_groups":httpGroups})
}

func (g *Gateway) responses(w http.ResponseWriter,httpRequest *http.Request,p Principal,id string) {
    g.inference(w,httpRequest,p,id,decodeRequest,false)
}

func (g *Gateway) chatCompletions(w http.ResponseWriter,httpRequest *http.Request,p Principal,id string) {
    g.inference(w,httpRequest,p,id,decodeChatRequest,true)
}

func (g *Gateway) inference(w http.ResponseWriter,httpRequest *http.Request,p Principal,id string,decode func([]byte)(Request,error),chat bool) {
    if httpRequest.Method!="POST" { fail(w,405,"method_not_allowed");return }
    if httpRequest.URL.RawQuery!="" { fail(w,400,"unsupported_query");return }
    media:=strings.TrimSpace(strings.Split(httpRequest.Header.Get("Content-Type"),";")[0])
    if media!="application/json" { fail(w,400,"invalid_content_type");return }
    data,err:=io.ReadAll(http.MaxBytesReader(w,httpRequest.Body,maxBody))
    if err!=nil { var sizeError *http.MaxBytesError;if errors.As(err,&sizeError) { fail(w,413,"payload_too_large") }else{fail(w,400,"invalid_body")};return }
    request,err:=decode(data)
    if err!=nil { fail(w,400,"invalid_request");return }
    if observed,ok:=w.(*observedWriter);ok {
        if _,configured:=g.config.Models[request.Model];configured { observed.record.Model=request.Model }
        if err=g.store.PutHTTPRequest(observed.record);err!=nil { fail(w,500,"storage_error");return }
    }
    if !p.allows(request.Model) { fail(w,400,"model_not_allowed");return }
    code:=g.acquire(p)
    if code!="" { status:=429;if code=="capacity_error" { status=503 };fail(w,status,code);return }
    defer g.release(p)
    ctx,cancel:=context.WithTimeout(httpRequest.Context(),time.Duration(g.config.TimeoutSeconds)*time.Second)
    defer cancel()
    stop:=context.AfterFunc(g.ctx,cancel);defer stop()
    created:=time.Now().UTC()
    stream:=&eventStream{writer:w,request:request,id:id,created:created,texts:[]string{}}
    var emit func(string)error
    deliveryError:=func(err error) {
        if err!=nil { if observed,ok:=w.(*observedWriter);ok { observed.deliveryFailed=true } }
    }
    if request.Stream { emit=func(text string)error { err:=stream.text(text);deliveryError(err);return err } }
    result:=executeApp(ctx,g.config,g.store,g.appPool,p,request,id,created,emit)
    if observed,ok:=w.(*observedWriter);ok {
        observed.record.State=result.State;observed.record.ErrorCode=result.Code;observed.record.ExecutionStarted=result.Started
        if !result.Started { observed.record.State="rejected" }
    }
    if request.Stream&&stream.started {
		if err:=stream.finishText();err!=nil { deliveryError(err) }
        status:="completed";event:="response.completed"
        if result.State!="succeeded" { status="failed";event="response.failed" }
        deliveryError(stream.send(event,map[string]any{"response":responseObject(id,created,request,status,result.Texts,result.ToolCall,result.Usage,result.Code)}))
        return
    }
    if result.State!="succeeded" { fail(w,result.HTTP,result.Code);return }
    if request.Stream {
        if err=stream.begin();err!=nil { deliveryError(err);return }
        deliveryError(stream.send("response.completed",map[string]any{"response":responseObject(id,created,request,"completed",result.Texts,result.ToolCall,result.Usage,"")}));return
    }
    if chat { writeJSON(w,200,chatCompletionObject(id,created,request,result.Texts,result.Usage));return }
    writeJSON(w,200,responseObject(id,created,request,"completed",result.Texts,result.ToolCall,result.Usage,""))
}

type eventStream struct {
    writer http.ResponseWriter
    request Request
    id string
    created time.Time
    sequence int
    started bool
    texts []string
	textActive bool
	textFinished bool
	textBuffer strings.Builder
	textItemID string
}

func (s *eventStream) send(kind string,data map[string]any) error {
    controller:=http.NewResponseController(s.writer)
    if err:=controller.SetWriteDeadline(time.Now().Add(5*time.Second));err!=nil { return err }
    data["type"]=kind;data["sequence_number"]=s.sequence;s.sequence++
    encoded,err:=json.Marshal(data);if err!=nil { return err }
    if _,err=io.WriteString(s.writer,"event: "+kind+"\ndata: "+string(encoded)+"\n\n");err!=nil { return err }
    return controller.Flush()
}

func (s *eventStream) begin() error {
    if s.started { return nil }
    s.writer.Header().Set("Content-Type","text/event-stream");s.writer.Header().Set("Cache-Control","no-cache");s.writer.Header().Set("X-Accel-Buffering","no")
    s.started=true
    for _,kind:=range []string{"response.created","response.in_progress"} { if err:=s.send(kind,map[string]any{"response":responseObject(s.id,s.created,s.request,"in_progress",nil,nil,nil,"")});err!=nil { return err } }
    return nil
}

func (s *eventStream) text(text string) error {
    if err:=s.begin();err!=nil { return err }
	if s.textFinished { return errors.New("stream text already finalized") }
	if !s.textActive {
		s.textActive=true;s.textItemID=s.id+"_msg_0"
		if err:=s.send("response.output_item.added",map[string]any{"output_index":0,"item":map[string]any{"id":s.textItemID,"type":"message","role":"assistant","status":"in_progress","content":[]any{}}});err!=nil{return err}
		if err:=s.send("response.content_part.added",map[string]any{"item_id":s.textItemID,"output_index":0,"content_index":0,"part":textPart("")});err!=nil{return err}
	}
	s.textBuffer.WriteString(text)
	return s.send("response.output_text.delta",map[string]any{"item_id":s.textItemID,"output_index":0,"content_index":0,"delta":text,"logprobs":[]any{}})
}

func (s *eventStream) finishText() error {
	if !s.textActive||s.textFinished { return nil }
	text:=s.textBuffer.String();s.textFinished=true;s.texts=[]string{text}
	for _,event:=range []struct{kind string;data map[string]any}{
		{"response.output_text.done",map[string]any{"item_id":s.textItemID,"output_index":0,"content_index":0,"text":text,"logprobs":[]any{}}},
		{"response.content_part.done",map[string]any{"item_id":s.textItemID,"output_index":0,"content_index":0,"part":textPart(text)}},
		{"response.output_item.done",map[string]any{"output_index":0,"item":message(s.textItemID,text,"completed")}},
	} { if err:=s.send(event.kind,event.data);err!=nil{return err} }
	return nil
}
