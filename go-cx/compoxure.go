package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/coocood/freecache"
	"github.com/valyala/fasthttp"
)

type compoxur struct {
	sync.Mutex
	cache   *freecache.Cache
	varSet  map[string]string
	backSet []backend
}

func (cx *compoxur) handle(ctx *fasthttp.RequestCtx) {
	cx.defineVars(ctx)
	ctx.Write(cx.processBack(ctx.getVal("url:path", "*")))
}

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

	cx.varSet["user:agent"] = string(ctx.UserAgent())
}

func (cx *compoxur) getVar(key, def string) string {
	cx.RLock()
	defer cx.RUnlock()

	if val, ok := cx.varSet[key]; ok && len(val) > 0 {
		return val
	}
	return def
}

func (cx *compoxur) processBack(path string) []byte {
	for _, b := range cx.backSet {
		if b.regexp == "*" || strings.Contains(path, b.regexp) {
			return b.process(cx)
		}
	}
	return []byte{}
}

func (cx *compoxure) parseHtml(html []byte, done chan struct{}) []byte {
	close(done)
	tpl := append([]byte{}, html...)
	//read till <div -> send to process fragment, add placeholder
	//anylize teg attributes, define cx-
	//parse parameters from attributes
	//make operation
	// <- return result, insert to placeholder

}
