package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/coocood/freecache"
	"github.com/valyala/fasthttp"
)

type backend struct {
	regexp string
	target string
	host   string
	//e.g. 1s = 1000, 1m = 60*1000 etc.
	//The valid values are 1s, 1m, 1h, 1d.
	//If you do not provide a suffix it assumes ms
	ttl          string
	timeout      string
	quietFailure bool
	dontPassUrl  bool
	contentTypes []string
	headers      []string
	chackeKey    string
	noCache      bool
}

type compoxur struct {
	sync.Mutex
	cache   *freecache.Cache
	varSet  map[string]string
	backSet []backend
}

func (cx *compoxur) processHandle(ctx *fasthttp.RequestCtx) {

}

/*
{ 'param:resourceId': '123456',
  'url:protocol': 'http:',
  'url:slashes': true,
  'url:auth': null,
  'url:host': 'localhost:5000',
  'url:port': '5000',
  'url:hostname': 'localhost',
  'url:hash': null,
  'url:search': '?param=true',
  'url:query': 'param=true',
  'url:pathname': '/resource/123456',
  'url:path': '/resource/123456?param=true',
  'url:href': 'http://localhost:5000/resource/123456?param=true',
  'cookie:example': '12345',
  'header:host': 'localhost:5000',
  'header:connection': 'keep-alive',
  'header:accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/ /*;q=0.8',
  'header:user-agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/36.0.1985.125 Safari/537.36',
  'header:referer': 'http://localhost:5000/',
  'header:accept-encoding': 'gzip,deflate,sdch',
  'header:accept-language': 'en-GB,en-US;q=0.8,en;q=0.6',
  'header:cookie': 'example=12345',
  'server:local': 'http://localhost:5001',
  'env:name': 'development',
  'user:userId': '_',
  'device:type': 'phone'
  }
Request method is "GET"
RequestURI is "/some/backend.html?test=one&and=one_more"
Requested path is "/some/backend.html"
Host is "localhost:8080"
Query string is "test=one&and=one_more"
*/
func (cx *compoxur) defineVars(ctx *fasthttp.RequestCtx) {
	cx.Lock()
	defer cx.Unlock()

	//url
	cx.varSet["url:protocol"] = "http"
	if ctx.IsTLS() {
		cx.varSet["url:protocol"] = "https"
	}
	cx.varSet["url:host"] = string(ctx.Host())
	hostParts := strings.Split(cx.varSet["url:host"], ":")
	cx.varSet["url:hostname"] = hostParts[0]
	if len(hostParts) >= 2 {
		cx.varSet["url:port"] = hostParts[1]
	}

	qArgs := ctx.QueryArgs()
	cx.varSet["url:query"] = string(qArgs.QueryString())
	cx.varSet["url:search"] = fmt.Sprintf("?%s", cx.varSet["url:query"])
	qArgs.VisitAll(func(key, value []byte) {
		cx.varSet[fmt.Sprintf("query:%s", string(key))] = string(value)
	})

	cx.varSet["url:pathname"] = string(ctx.Path())
	cx.varSet["url:path"] = string(ctx.RequestURI())
	cx.varSet["url:href"] = fmt.Sprintf("%s://%s/%s", cx.varSet["url:protocol"], cx.varSet["url:host"], cx.varSet["url:path"])
}

func (cx *compoxur) selectBeck(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	for _, b := range cx.backSet {
		if b.regexp == "*" || strings.Contains(path, b.regexp) {
			break
		}
	}
}
