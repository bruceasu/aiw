package main

import (
    "bytes"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "errors"
    "io"
    "strings"
    "time"
    "unicode/utf8"
)

type Request struct {
    Model string `json:"model"`
    Input json.RawMessage `json:"input"`
    Instructions string `json:"instructions,omitempty"`
    Stream bool `json:"stream,omitempty"`
    Store bool `json:"store,omitempty"`
    Background bool `json:"background,omitempty"`
    Tools []FunctionTool `json:"tools,omitempty"`
    Include []json.RawMessage `json:"include,omitempty"`
    ToolChoice string `json:"tool_choice,omitempty"`
    ParallelToolCalls *bool `json:"parallel_tool_calls,omitempty"`
    Text *struct { Format struct { Type string `json:"type"` } `json:"format"` } `json:"text,omitempty"`
    prompt string
    jsonObject bool
}

type FunctionTool struct {
    Type string `json:"type"`
    Name string `json:"name"`
    Description string `json:"description,omitempty"`
    Parameters json.RawMessage `json:"parameters"`
    Strict *bool `json:"strict,omitempty"`
}

type FunctionCall struct {
    ID string `json:"id"`
    Type string `json:"type"`
    Status string `json:"status"`
    CallID string `json:"call_id"`
    Name string `json:"name"`
    Arguments string `json:"arguments"`
}

func validToolName(name string) bool {
    if len(name)==0||len(name)>64 { return false }
    for _,ch:=range name { if !(ch>='a'&&ch<='z'||ch>='A'&&ch<='Z'||ch>='0'&&ch<='9'||ch=='_'||ch=='-') { return false } }
    return true
}

func validJSONObject(data []byte) bool {
    var object map[string]json.RawMessage
    return strictDecode(data,&object)==nil&&object!=nil
}

func rejectDuplicates(dec *json.Decoder) error {
    token,err:=dec.Token(); if err!=nil { return err }
    delim,ok:=token.(json.Delim); if !ok { return nil }
    switch delim {
    case '{':
        keys:=map[string]bool{}
        for dec.More() { key,e:=dec.Token(); if e!=nil { return e }; name,ok:=key.(string); if !ok||keys[name] { return errors.New("duplicate JSON key") }; keys[name]=true; if e=rejectDuplicates(dec);e!=nil { return e } }
    case '[':
        for dec.More() { if err=rejectDuplicates(dec);err!=nil { return err } }
    default: return errors.New("invalid JSON delimiter")
    }
    _,err=dec.Token(); return err
}

func strictDecode(data []byte, target any) error {
    if !utf8.Valid(data) { return errors.New("invalid UTF-8") }
    duplicate:=json.NewDecoder(bytes.NewReader(data))
    if err:=rejectDuplicates(duplicate);err!=nil { return err }
    if _,err:=duplicate.Token();err!=io.EOF { return errors.New("trailing JSON") }
    dec:=json.NewDecoder(bytes.NewReader(data)); dec.DisallowUnknownFields()
    return dec.Decode(target)
}

func decodeRequest(data []byte) (Request,error) {
    var r Request
    if err:=strictDecode(data,&r);err!=nil { return r,errors.New("invalid or unsupported request fields") }
    var fields map[string]json.RawMessage
    _=json.Unmarshal(data,&fields)
    for key,value:=range fields { if bytes.Equal(bytes.TrimSpace(value),[]byte("null")) { return r,errors.New("null request fields are unsupported: "+key) } }
    if r.Model==""||r.Store||r.Background||len(r.Include)>0 { return r,errors.New("unsupported model or execution options") }
    if len(r.Tools)>0 {
        if len(r.Tools)>16||r.Stream { return r,errors.New("streaming function tools are unsupported") }
        if r.ToolChoice=="" { r.ToolChoice="auto" }
        if r.ToolChoice!="auto"&&r.ToolChoice!="none"&&r.ToolChoice!="required" { return r,errors.New("unsupported tool choice") }
        if r.ParallelToolCalls!=nil&&*r.ParallelToolCalls { return r,errors.New("parallel tool calls are unsupported") }
        names:=map[string]bool{}
        for _,tool:=range r.Tools {
            if tool.Type!="function"||!validToolName(tool.Name)||names[tool.Name]||!validJSONObject(tool.Parameters) { return r,errors.New("unsupported function tool") }
            if tool.Strict!=nil&&*tool.Strict&&validateFunctionSchema(tool.Parameters)!=nil { return r,errors.New("unsupported strict function schema") }
            names[tool.Name]=true
        }
        if r.Text!=nil { return r,errors.New("text format with function tools is unsupported") }
    } else if r.ToolChoice!=""||r.ParallelToolCalls!=nil { return r,errors.New("tool options require tools") }
    if r.Text!=nil {
        switch r.Text.Format.Type { case "text": case "json_object":r.jsonObject=true; default:return r,errors.New("unsupported text format") }
    }
    var plain string
    if json.Unmarshal(r.Input,&plain)==nil { r.prompt=plain } else {
        var messages []json.RawMessage
        if err:=strictDecode(r.Input,&messages);err!=nil||len(messages)==0 { return r,errors.New("input must contain text") }
        var parts []string
        calls:=map[string]bool{}
        for _,raw:=range messages {
            var item struct { Type string `json:"type"` }
            if json.Unmarshal(raw,&item)!=nil { return r,errors.New("invalid input item") }
            if item.Type=="function_call" {
                var call struct { Type string `json:"type"`; CallID string `json:"call_id"`; Name string `json:"name"`; Arguments string `json:"arguments"`; ID string `json:"id,omitempty"`; Status string `json:"status,omitempty"` }
                if len(r.Tools)==0||strictDecode(raw,&call)!=nil||call.CallID==""||len(call.CallID)>128||!validToolName(call.Name)||!validJSONObject([]byte(call.Arguments))||calls[call.CallID] { return r,errors.New("invalid function call input") }
                calls[call.CallID]=true
                parts=append(parts,"assistant function_call "+call.Name+" ("+call.CallID+"):\n"+call.Arguments)
                continue
            }
            if item.Type=="function_call_output" {
                var output struct { Type string `json:"type"`; CallID string `json:"call_id"`; Output string `json:"output"`; ID string `json:"id,omitempty"`; Status string `json:"status,omitempty"` }
                if strictDecode(raw,&output)!=nil||!calls[output.CallID] { return r,errors.New("unmatched function call output") }
                delete(calls,output.CallID)
                parts=append(parts,"tool result ("+output.CallID+"):\n"+output.Output)
                continue
            }
            var m struct { Role string `json:"role"`; Content json.RawMessage `json:"content"`; Type string `json:"type,omitempty"`; ID string `json:"id,omitempty"`; Status string `json:"status,omitempty"`; Phase string `json:"phase,omitempty"` }
            if strictDecode(raw,&m)!=nil||(m.Role!="user"&&m.Role!="assistant")||(m.Type!=""&&m.Type!="message") { return r,errors.New("unsupported input role or type") }
            var text string
            if json.Unmarshal(m.Content,&text)!=nil {
                var content []struct { Type string `json:"type"`; Text string `json:"text"`; Annotations []json.RawMessage `json:"annotations,omitempty"`; Logprobs []json.RawMessage `json:"logprobs,omitempty"` }
                if err:=strictDecode(m.Content,&content);err!=nil||len(content)==0 { return r,errors.New("unsupported input content") }
                for _,part:=range content {
                    if part.Type!="input_text"&&!(m.Role=="assistant"&&part.Type=="output_text") { return r,errors.New("unsupported input content type") }
                    if len(part.Annotations)>0||len(part.Logprobs)>0 { return r,errors.New("input annotations are unsupported") }
                    text+=part.Text
                }
            }
            parts=append(parts,m.Role+":\n"+text)
        }
        r.prompt=strings.Join(parts,"\n\n")
    }
    toolBytes:=0
    if len(r.Tools)>0 { toolsJSON,_:=json.Marshal(r.Tools);toolBytes=len(toolsJSON) }
    if strings.TrimSpace(r.prompt)==""||len(r.prompt)+len(r.Instructions)+toolBytes>maxText { return r,errors.New("input is empty or exceeds 64 KiB") }
    return r,nil
}

type Usage struct {
    InputTokens int64 `json:"input_tokens"`
    OutputTokens int64 `json:"output_tokens"`
    TotalTokens int64 `json:"total_tokens"`
    InputDetails any `json:"input_tokens_details"`
    OutputDetails any `json:"output_tokens_details"`
}

func newID() string {
    var raw [16]byte
    if _,err:=rand.Read(raw[:]);err!=nil { panic("operating system random source unavailable") }
    return "resp_"+hex.EncodeToString(raw[:])
}

func textPart(text string) map[string]any { return map[string]any{"type":"output_text","text":text,"annotations":[]any{},"logprobs":[]any{}} }
func message(id,text,status string) map[string]any { return map[string]any{"id":id,"type":"message","role":"assistant","status":status,"content":[]any{textPart(text)}} }
func responseObject(id string, created time.Time, r Request, status string, texts []string, call *FunctionCall, usage *Usage, code string) map[string]any {
    output:=[]any{}
    for i,text:=range texts { output=append(output,message(id+"_msg_"+intString(i),text,"completed")) }
    if call!=nil { output=append(output,call) }
    var responseError any
    if code!="" { responseError=map[string]any{"code":code,"message":"Gateway execution failed"} }
    format:="text";if r.jsonObject { format="json_object" }
    var instructions any
    if r.Instructions!="" { instructions=r.Instructions }
    tools:=any([]any{});choice:="none"
    if len(r.Tools)>0 { tools=r.Tools;choice=r.ToolChoice }
    return map[string]any{"id":id,"object":"response","created_at":created.Unix(),"status":status,"model":r.Model,"output":output,"error":responseError,"incomplete_details":nil,"usage":usage,"instructions":instructions,"tools":tools,"tool_choice":choice,"parallel_tool_calls":false,"text":map[string]any{"format":map[string]any{"type":format}},"metadata":map[string]any{},"store":false,"background":false,"previous_response_id":nil,"temperature":nil,"top_p":nil,"max_output_tokens":nil,"reasoning":map[string]any{"effort":nil,"summary":nil},"truncation":"disabled"}
}
