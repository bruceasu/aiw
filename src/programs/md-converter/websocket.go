package main

// A deliberately narrow, sequential RFC6455 transport for local Chrome CDP.
// No proxy, TLS, compression, remote endpoints or parallel callers are supported.
import (
    "bufio"
    "crypto/rand"
    "crypto/sha1"
    "encoding/base64"
    "encoding/binary"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "strings"
    "time"
)

type cdp struct {
    conn net.Conn
    input *bufio.Reader
    id int
    failure error
}

func connectCDP(address string, deadline time.Time) (*cdp, error) {
    u, err := url.Parse(address)
    if err != nil || u.Scheme != "ws" || u.Hostname() != "127.0.0.1" || u.User != nil { return nil, errors.New("non-local CDP endpoint rejected") }
    conn, err := net.DialTimeout("tcp", u.Host, 3*time.Second)
    if err != nil { return nil, err }
    ok := false
    defer func() { if !ok { _ = conn.Close() } }()
    _ = conn.SetDeadline(deadline)
    nonce := make([]byte, 16)
    if _, err = rand.Read(nonce); err != nil { return nil, err }
    key := base64.StdEncoding.EncodeToString(nonce)
    req := &http.Request{Method:"GET", URL:u, Host:u.Host, Header:http.Header{
        "Upgrade":{"websocket"}, "Connection":{"Upgrade"}, "Sec-WebSocket-Version":{"13"}, "Sec-WebSocket-Key":{key},
    }}
    if err = req.Write(conn); err != nil { return nil, err }
    reader := bufio.NewReader(conn)
    response, err := http.ReadResponse(reader, req)
    if err != nil { return nil, err }
    digest := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
    if response.StatusCode != 101 || !strings.EqualFold(response.Header.Get("Upgrade"), "websocket") || response.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(digest[:]) {
        return nil, errors.New("invalid local WebSocket handshake")
    }
    ok = true
    return &cdp{conn:conn, input:reader}, nil
}

func (c *cdp) frame(op byte, body []byte) error {
    header := []byte{0x80 | op}
    switch n := len(body); {
    case n < 126: header = append(header, 0x80 | byte(n))
    case n <= 65535: header = append(header, 0x80|126, byte(n>>8), byte(n))
    default: header = append(header, 0x80|127); header = binary.BigEndian.AppendUint64(header, uint64(n))
    }
    mask := make([]byte, 4)
    if _, err := rand.Read(mask); err != nil { return err }
    header = append(header, mask...)
    payload := make([]byte, len(body))
    for i, v := range body { payload[i] = v ^ mask[i%4] }
    if _, err := io.Copy(c.conn, strings.NewReader(string(header))); err != nil { return err }
    _, err := io.Copy(c.conn, strings.NewReader(string(payload)))
    return err
}

func (c *cdp) message() ([]byte, error) {
    var message []byte
    started := false
    for {
        header := make([]byte, 2)
        if _, err := io.ReadFull(c.input, header); err != nil { return nil, err }
        if header[0]&0x70 != 0 || header[1]&0x80 != 0 { return nil, errors.New("unsupported WebSocket frame") }
        n := uint64(header[1]&127)
        if n == 126 {
            b := make([]byte,2); if _,err:=io.ReadFull(c.input,b);err!=nil{return nil,err}; n=uint64(binary.BigEndian.Uint16(b))
        } else if n == 127 {
            b := make([]byte,8); if _,err:=io.ReadFull(c.input,b);err!=nil{return nil,err}; n=binary.BigEndian.Uint64(b)
        }
        if n > 128*1024*1024 || uint64(len(message))+n > 128*1024*1024 { return nil, errors.New("CDP message exceeds 128 MiB") }
        body := make([]byte,int(n)); if _,err:=io.ReadFull(c.input,body);err!=nil{return nil,err}
        op := header[0]&15
        if op >= 8 {
            if header[0]&0x80 == 0 || n > 125 { return nil, errors.New("invalid control frame") }
            switch op {
            case 8: return nil, io.EOF
            case 9: if err:=c.frame(10,body);err!=nil{return nil,err}
            case 10:
            default: return nil, errors.New("unknown WebSocket control frame")
            }
            continue
        }
        if (op == 1 && started) || (op == 0 && !started) || (op != 0 && op != 1) { return nil, errors.New("invalid CDP text frame sequence") }
        started = true
        message = append(message, body...)
        if header[0]&0x80 != 0 { return message, nil }
    }
}

func (c *cdp) call(method string, params any) (json.RawMessage,error) {
    c.id++
    payload,err:=json.Marshal(map[string]any{"id":c.id,"method":method,"params":params})
    if err!=nil{return nil,err}
    if err=c.frame(1,payload);err!=nil{return nil,err}
    for {
        body,err:=c.message();if err!=nil{return nil,err}
        var response struct { ID int `json:"id"`; Method string `json:"method"`; Error json.RawMessage `json:"error"`; Result json.RawMessage `json:"result"`; Params json.RawMessage `json:"params"` }
        if err=json.Unmarshal(body,&response);err!=nil{return nil,err}
        switch response.Method {
        case "Network.loadingFailed", "Runtime.exceptionThrown":
            c.failure = fmt.Errorf("render event %s: %.1500s",response.Method,response.Params)
        }
        if response.ID != c.id {continue}
        if len(response.Error)>0{return nil,fmt.Errorf("%s: %s",method,response.Error)}
        if c.failure!=nil{return nil,c.failure}
        return response.Result,nil
    }
}
