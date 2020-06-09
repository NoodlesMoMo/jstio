/*
	stupid http client for sogou-server-team.
*/

package requests

// issues: https://github.com/valyala/fasthttp/issues/318

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"net/url"
	"os"
	"sync"
	"time"
)

const (
	envoyEndpoint = `http://127.0.0.1/`
	localhost     = `127.0.0.1`
)

const (
	HTTP_METHOD_GET    = `GET`
	HTTP_METHOD_POST   = `POST`
	HTTP_METHOD_PUT    = `PUT`
	HTTP_METHOD_DELETE = `DELETE`
)

type RequestOption func(options *Options)

type Options struct {
	disableProxy bool
	maxConn      int
	readTimeout  time.Duration
	writeTimeout time.Duration
	maxIdle      time.Duration
}

func WithDisableProxy() RequestOption {
	return func(options *Options) {
		options.disableProxy = true
	}
}

func WithMaxConn(cnt int) RequestOption {
	return func(options *Options) {
		options.maxConn = cnt
	}
}

func init() {
	disableProxyEnv := os.Getenv("DISABLE_ENVOY_PROXY")
	if disableProxyEnv == "1" || disableProxyEnv == "true" {
		disableProxy = true
	}
}

var (
	disableProxy = false

	clients = clientsPool{
		lock:  sync.RWMutex{},
		cache: make(map[string]*fasthttp.HostClient),
	}
)

type clientsPool struct {
	lock  sync.RWMutex
	cache map[string]*fasthttp.HostClient
}

func (cp *clientsPool) Get(vHost string, options ...RequestOption) *fasthttp.HostClient {
	cp.lock.RLock()
	cli, ok := cp.cache[vHost]
	if ok {
		cp.lock.RUnlock()
		return cli
	}
	cp.lock.RUnlock()

	cli = cp.new(vHost, options...)

	return cli
}

func (cp *clientsPool) new(vHost string, options ...RequestOption) *fasthttp.HostClient {
	cp.lock.Lock()

	// double check
	if cli, ok := cp.cache[vHost]; ok {
		cp.lock.Unlock()
		return cli
	}

	defaultOptions := &Options{
		disableProxy: disableProxy,
		maxConn:      8192,
		readTimeout:  3 * time.Second,
		writeTimeout: 3 * time.Second,
	}
	for _, opt := range options {
		opt(defaultOptions)
	}

	addr := localhost
	if defaultOptions.disableProxy {
		addr = vHost
	}

	cli := &fasthttp.HostClient{
		Addr:         addr,
		MaxConns:     8192,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	cp.cache[vHost] = cli

	cp.lock.Unlock()

	return cli
}

func HTTPGet(path string, params url.Values, vHost string, options ...RequestOption) ([]byte, error) {
	return HTTPDo(HTTP_METHOD_GET, path, params, vHost, nil, options...)
}

func HTTPPost(path string, params url.Values, vHost string, body []byte, options ...RequestOption) ([]byte, error) {
	return HTTPDo(HTTP_METHOD_POST, path, params, vHost, body, options...)
}

func HTTPDo(method string, path string, params url.Values, vHost string, body []byte, options ...RequestOption) ([]byte, error) {
	var (
		err error
	)

	uri := path
	if path[0] != '/' {
		uri = "/" + path
	}

	if params != nil {
		c := path[len(path)-1]
		if c == '?' {
			uri += params.Encode()
		} else {
			uri += "?" + params.Encode()
		}
	}

	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(uri)
	req.Header.SetHost(vHost)
	if body != nil {
		req.SetBody(body)
	}
	req.Header.SetMethod(method)

	err = clients.Get(vHost, options...).Do(req, resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("http code: %d, domain: %s, uri: %s", resp.StatusCode(), vHost, uri)
	}

	respBody := make([]byte, len(resp.Body()))
	copy(respBody, resp.Body())

	return respBody, nil
}
