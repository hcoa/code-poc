package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/coocood/freecache"
	"github.com/valyala/fasthttp"
)

type compoxur struct {
	sync.RWMutex
	cache   *freecache.Cache
	varSet  map[string]string
	backSet []backend
}

func (cx *compoxur) handle(ctx *fasthttp.RequestCtx) {
	cx.defineVars(ctx)
	ctx.Write(cx.processBack(cx.getVar("url:path", "*")))
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
	cx.varSet["url:href"] = fmt.Sprintf("%s://%s/%s", cx.varSet["url:protocol"],
		cx.varSet["url:host"], cx.varSet["url:path"])

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
	fmt.Println("Compoxure->processBack ", path)
	for _, b := range cx.backSet {
		if b.regexp == "*" || strings.Contains(path, b.regexp) {
			fmt.Println(b.host, "run process")
			return b.process(cx)
		}
	}
	return []byte{}
}

func (cx *compoxur) parseHtml(html []byte, b *backend, fCh chan *fragment) ([]byte, int) {
	fmt.Printf("Compoxur->parseHtml: \ninput: %q\nbackend: %s\n", html, b.host)
	tpl := append([]byte{}, html...)
	r := bytes.NewReader(html)
	var err error
	fgCnt := 0
	var fg []byte
	var str *skipTillReader
	var rtr *readTillReader
	for err == nil {
		str = newSkipTillReader(r, []byte(`<div`))
		rtr = newReadTillReader(str, []byte(`</div>`))
		fg, err = ioutil.ReadAll(rtr)
		fmt.Printf("Read fragment: %q\nerr: %v\n", fg, err)
		if len(fg) == 0 {
			break
		}
		//replace the fragment code with placeholder in tpl
		tpl = append(
			tpl[:(str.cnt-4)],
			append(
				[]byte(fmt.Sprintf("{{fragment%d}}", fgCnt)),
				tpl[(rtr.cnt):]...,
			)...)
		fCh <- &fragment{
			id:  fgCnt,
			src: fg,
			b:   b,
		}
		fgCnt++
		fmt.Println("fragment count:", fgCnt)
	}
	return tpl, fgCnt

}
