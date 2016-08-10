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

/*
func (cx *compoxur) setVar2Html(html []byte) []byte {
	fmt.Println("Compoxure->setVar2Html, input: ", html)
	ob := []byte(`{{`)
	cb := []byte(`}}`)
	var pob, pcb, ocp int
	var key, val []byte
	var isCondition bool
	for {
		pob = bytes.Index(html[pob:], ob)
		pcb = bytes.Index(html[pob:], cb)
		if pob == -1 || pcb == -1 {
			break
		}
		isCondition = false
		key = html[(pob + 2):(pcb - 1)]
		fmt.Printf("key: %q\n", key)
		if key[0] == '#' {
			key = key[1:]
			isCondition = true
		}
		val = []byte(cx.getVar(string(key), ""))
		if len(val) > 0 && !isCondition {
			html = append(html[pob:], append(val, html[:(pcb+2)]...)...)
			fmt.Printf("insert %q with key %q to html\n", val, key)
		}
		//remove close value block
		if isCondition {
			ocp = pob //save beginning of condition block
			pob = bytes.Index(html[pob:], ob)
			//if value not empty condition is TRUE
			if len(val) > 0 {
				val = html[(pcb + 2):pob]
				fmt.Println("val in condition block: ", val)
			}
			pcb = bytes.Index(html[pob:], cb)
			if len(val) > 0 {
				html = append(html[ocp:], append(val, html[(pcb+2):]...)...)
			} else {
				html = append(html[ocp:], html[(pcb+2):]...)
			}
		}
	}
	return html
}
*/

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
	openTeg := []byte(`<div`)
	closeTeg := []byte(`</div>`)
	var str *skipTillReader
	var rtr *readTillReader
	for err == nil {
		str = newSkipTillReader(r, openTeg)
		rtr = newReadTillReader(str, closeTeg)
		fg, err = ioutil.ReadAll(rtr)
		fmt.Printf("Read fragment: %q\nerr: %v\n", fg, err)
		if len(fg) == 0 || err != nil {
			break
		}
		//replace the fragment code with placeholder in tpl
		tpl = append(
			tpl[:(str.cnt-len(openTeg))],
			append(
				[]byte(fmt.Sprintf("{{fragment%d}}", fgCnt)),
				tpl[(rtr.cnt+len(openTeg)+len(closeTeg)-1):]...,
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

func (cx *compoxur) parseTeg(src []byte) []byte {
	endTegPos := bytes.IndexByte(src, '>')
	teg := src[:endTegPos]
	fmt.Printf("Compoxure->parseTeg: %q\n", teg)
	//attrSet := bytes.Split(teg, []byte{' '})
	return src
}

var (
	cxUrl      = []byte(`cx-url`)
	cxCacheKey = []byte(`cx-cache-key`)
	cxCacheTtl = []byte(`cx-cache-ttl`)
	cxTimeout  = []byte(`cx-timeout`)
)

type tegCxAttr struct {
	cxUrl      []byte
	cxCacheKey []byte
	cxCacheTtl []byte
	cxTimeout  []byte
}

func newTegCxAttr(attrSet [][]byte) *tegCxAttr {
	var pos int
	tca := &tegCxAttr{}
	for _, attr := range attrSet {
		pos = bytes.IndexByte(attr, '=')
		switch string(attr[:pos]) {
		case string(cxUrl):
			tca.cxUrl = attr[(pos + 1):]
		case string(cxTimeout):
			tca.cxTimeout = attr[(pos + 1):]
		case string(cxCacheKey):
			tca.cxCacheKey = attr[(pos + 1):]
		case string(cxCacheTtl):
			tca.cxCacheTtl = attr[(pos + 1):]
		}
	}
	return tca
}
