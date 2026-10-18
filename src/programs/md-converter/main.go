package main

import (
    "crypto/sha256"
    "encoding/base64"
    "flag"
    "fmt"
    "html"
    "os"
    "path/filepath"
    "strings"
    "aiw-md-converter/internal/assets"
    "aiw-md-converter/internal/markdown"
)

func document(body string, includeMermaid bool) string {
    script:=assets.Mermaid
    // Prevent embedded JS from closing its HTML script container.
    script=strings.ReplaceAll(script,"</script","<\\/script")
    scriptHash:=sha256.Sum256([]byte(script))
    policy:="default-src 'none'; img-src data:; font-src data:; style-src 'unsafe-inline'; script-src 'sha256-"+base64.StdEncoding.EncodeToString(scriptHash[:])+"'; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"
    head:="<!doctype html><html><head><meta charset=\"utf-8\"><meta http-equiv=\"Content-Security-Policy\" content=\""+html.EscapeString(policy)+"\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Converted document</title><style>"+assets.CSS+"</style></head><body>"
    tail:=""
    if includeMermaid {tail="<script>"+script+"</script>"}
    // Standalone HTML renders Mermaid too; PDF awaits the same readiness explicitly.
    if includeMermaid {
        init:=`window.aiwMermaidReady=(async()=>{mermaid.initialize({startOnLoad:false,securityLevel:'strict'});await mermaid.run({querySelector:'.mermaid'});})();`
        sum:=sha256.Sum256([]byte(init))
        head=strings.Replace(head,"; connect-src", " 'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'; connect-src",1)
        tail+="<script>"+init+"</script>"
    }
    return head+body+tail+"</body></html>"
}

func main() {
    input:=flag.String("input","","Local MD or preprocessed passive HTML input")
    output:=flag.String("output","","Temporary output file")
    target:=flag.String("format","html","html, pdf or txt")
    flag.Parse()
    if err:=convert(*input,*output,*target);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
}

func convert(input,output,target string) error {
    if input==""||output==""||output=="-"{return fmt.Errorf("input and output files are required")}
    source,err:=os.ReadFile(input);if err!=nil{return err}
    ext:=strings.ToLower(filepath.Ext(input))
    if target=="txt" {
        if ext!=".md"&&ext!=".markdown"{return fmt.Errorf("TXT input must be Markdown")}
        return os.WriteFile(output,[]byte(markdown.Plain(source)),0600)
    }
    var body string
    var mermaid bool
    if ext==".md"||ext==".markdown" {
        rendered,e:=markdown.Render(source,markdown.Options{ImageRoot:filepath.Dir(input)})
        if e!=nil{return e};body="<main class=\"document\">"+string(rendered)+"</main>";mermaid=strings.Contains(body,"class=\"mermaid\"")
    } else if ext==".html"||ext==".htm" {
        // The Python plugin owns HTML parsing, resource embedding and rejection
        // of active content; this internal helper is not a general HTML CLI.
        body=string(source)
    } else {return fmt.Errorf("unsupported input extension")}
    page:=document(body,mermaid)
    switch target {
    case "html": return os.WriteFile(output,[]byte(page),0600)
    case "pdf": return printPDF(page,output)
    default: return fmt.Errorf("unsupported target format")
    }
}
