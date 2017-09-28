package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	c "golang.org/x/net/context"
)

var (
	UserAgent = "Golang z/rpc package"
	UserKey   = "X-User"
	reqidKey  = 0
)

var (
	ErrInvalidRequestURL = errors.New("invalid request url")
)

type Client struct {
	*http.Client
}

var (
	DefaultClient = Client{&http.Client{Transport: http.DefaultTransport}}
)

func newRequest(method, url1 string, body io.Reader) (req *http.Request, err error) {

	var host string

	// url1 = "-H <Host> http://<ip>[:<port>]/<path>"

	if strings.HasPrefix(url1, "-H") {
		url2 := strings.TrimLeft(url1[2:], " \t")
		pos := strings.Index(url2, " ")
		if pos <= 0 {
			return nil, ErrInvalidRequestURL
		}
		host = url2[:pos]
		url1 = strings.TrimLeft(url2[pos+1:], " \t")
	}

	req, err = http.NewRequest(method, url1, body)
	if err != nil {
		return
	}
	if host != "" {
		req.Host = host
	}
	return
}

// Do ...
func (r Client) Do(ctx c.Context, req *http.Request) (resp *http.Response, err error) {
	if ctx == nil {
		ctx = c.Background()
	}

	if reqid, ok := ctx.Value(reqidKey).(string); ok {
		req.Header.Set("X-Reqid", reqid)
	}

	if user, ok := ctx.Value(UserKey).(string); ok {
		req.Header.Set(UserKey, user)
	}

	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", UserAgent)
	}

	transport := r.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// avoid cancel() is called before Do(req), but isn't accurate
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	if tr, ok := getRequestCanceler(transport); ok { // support CancelRequest
		reqC := make(chan bool, 1)
		go func() {
			resp, err = r.Client.Do(req)
			reqC <- true
		}()
		select {
		case <-reqC:
		case <-ctx.Done():
			tr.CancelRequest(req)
			<-reqC
			err = ctx.Err()
		}
	} else {
		resp, err = r.Client.Do(req)
	}
	return
}

//DoRequest ...
func (r Client) DoRequest(ctx c.Context, method, url string) (resp *http.Response, err error) {
	req, err := newRequest(method, url, nil)
	if err != nil {
		return
	}
	return r.Do(ctx, req)
}

//DoRequestWith ...
func (r Client) DoRequestWith(ctx c.Context, method, url1 string, bodyType string, body io.Reader, bodyLength int) (resp *http.Response, err error) {

	req, err := newRequest(method, url1, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", bodyType)
	req.ContentLength = int64(bodyLength)
	return r.Do(ctx, req)
}

//DoRequestWithForm ...
func (r Client) DoRequestWithForm(ctx c.Context, method, url1 string, data map[string][]string) (resp *http.Response, err error) {

	msg := url.Values(data).Encode()
	if method == "GET" || method == "HEAD" || method == "DELETE" {
		if strings.ContainsRune(url1, '?') {
			url1 += "&"
		} else {
			url1 += "?"
		}
		return r.DoRequest(ctx, method, url1+msg)
	}
	return r.DoRequestWith(ctx, method, url1, "application/x-www-form-urlencoded", strings.NewReader(msg), len(msg))
}

//DoRequestWithJSON ...
func (r Client) DoRequestWithJSON(ctx c.Context, method, url1 string, data interface{}) (resp *http.Response, err error) {

	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
	return r.DoRequestWith(ctx, method, url1, "application/json", bytes.NewReader(msg), len(msg))
}

//ErrorInfo ...
type ErrorInfo struct {
	Err   string `json:"error,omitempty"`
	Key   string `json:"key,omitempty"`
	Reqid string `json:"reqid,omitempty"`
	Errno int    `json:"errno,omitempty"`
	Code  int    `json:"code"`
}

//ErrorDetail ...
func (r *ErrorInfo) ErrorDetail() string {
	msg, _ := json.Marshal(r)
	return string(msg)
}

func (r *ErrorInfo) Error() string {
	return r.Err
}

//RpcError ...
func (r *ErrorInfo) RpcError() (code, errno int, key, err string) {

	return r.Code, r.Errno, r.Key, r.Err
}

//HttpCode ...
func (r *ErrorInfo) HttpCode() int {

	return r.Code
}

func parseError(e *ErrorInfo, r io.Reader) {

	body, err1 := ioutil.ReadAll(r)
	if err1 != nil {
		e.Err = err1.Error()
		return
	}

	var ret struct {
		Err   string `json:"error"`
		Key   string `json:"key"`
		Errno int    `json:"errno"`
	}
	if json.Unmarshal(body, &ret) == nil && ret.Err != "" {
		// qiniu error msg style returns here
		e.Err, e.Key, e.Errno = ret.Err, ret.Key, ret.Errno
		return
	}
	e.Err = string(body)
}

// ResponseError ...
func ResponseError(resp *http.Response) (err error) {

	e := &ErrorInfo{
		Reqid: resp.Header.Get("X-Reqid"),
		Code:  resp.StatusCode,
	}
	if resp.StatusCode > 299 {
		if resp.ContentLength != 0 {
			ct, ok := resp.Header["Content-Type"]
			if ok && (strings.HasPrefix(ct[0], "application/json") || strings.HasPrefix(ct[0], "text/plain")) {
				parseError(e, resp.Body)
			}
		}
	}
	return e
}

// CallRet ...
func CallRet(ctx c.Context, ret interface{}, resp *http.Response) (err error) {
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode/100 == 2 {
		if ret != nil && resp.ContentLength != 0 {
			err = json.NewDecoder(resp.Body).Decode(ret)
			if err != nil {
				return
			}
		}
		if resp.StatusCode == 200 {
			return nil
		}
	}
	return ResponseError(resp)
}

// CallWithForm ...
func (r Client) CallWithForm(ctx c.Context, ret interface{}, method, url1 string, param map[string][]string) (err error) {
	resp, err := r.DoRequestWithForm(ctx, method, url1, param)
	if err != nil {
		return err
	}
	return CallRet(ctx, ret, resp)
}

// CallWithJSON ...
func (r Client) CallWithJSON(ctx c.Context, ret interface{}, method, url1 string, param interface{}) (err error) {
	resp, err := r.DoRequestWithJSON(ctx, method, url1, param)
	if err != nil {
		return err
	}
	return CallRet(ctx, ret, resp)
}

// CallWith ...
func (r Client) CallWith(ctx c.Context, ret interface{}, method, url1, bodyType string, body io.Reader, bodyLength int) (err error) {
	resp, err := r.DoRequestWith(ctx, method, url1, bodyType, body, bodyLength)
	if err != nil {
		return err
	}
	return CallRet(ctx, ret, resp)
}

//Call ...
func (r Client) Call(ctx c.Context, ret interface{}, method, url1 string) (err error) {

	resp, err := r.DoRequestWith(ctx, method, url1, "application/x-www-form-urlencoded", nil, 0)
	if err != nil {
		return err
	}
	return CallRet(ctx, ret, resp)
}

type requestCanceler interface {
	CancelRequest(req *http.Request)
}

type nestedObjectGetter interface {
	NestedObject() interface{}
}

func getRequestCanceler(tp http.RoundTripper) (rc requestCanceler, ok bool) {

	if rc, ok = tp.(requestCanceler); ok {
		return
	}

	p := interface{}(tp)
	for {
		getter, ok1 := p.(nestedObjectGetter)
		if !ok1 {
			return
		}
		p = getter.NestedObject()
		if rc, ok = p.(requestCanceler); ok {
			return
		}
	}
}
