package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/coocood/freecache"
	"github.com/hoisie/mustache"
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
	tca := &tegCxAttr{}
	tca.cxUrl = ctx.RequestURI()
	ctx.Write(cx.processBack(tca))
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

func (cx *compoxur) processBack(tca *tegCxAttr) []byte {
	fmt.Println("Compoxure->processBack ", string(tca.cxUrl))
	for _, b := range cx.backSet {
		if b.regexp[0] == '*' && len(b.regexp) == 1 || bytes.Contains(tca.cxUrl, b.regexp) {
			fmt.Println(b.host, "run process")
			return b.process(cx, tca)
		}
	}
	return []byte{}
}

func (cx *compoxur) parseHtml(html []byte, b *backend, fCh chan *fragment) ([]byte, int) {
	fmt.Printf("Compoxur->parseHtml: \ninput: %q\nlen: %d\nbackend: %s\n", html, len(html), b.host)
	tpl := append([]byte{}, html...)
	r := bytes.NewReader(html)
	var err error
	fgCnt := 0
	var fg []byte
	openTeg := []byte(`<div `)
	closeTeg := []byte(`</div>`)
	var str *skipTillReader
	var rtr *readTillReader
	var sPos, ePos int
	for err == nil {
		str = newSkipTillReader(r, openTeg)
		rtr = newReadTillReader(str, closeTeg)
		fg, err = ioutil.ReadAll(rtr)
		//empty or dive without cx-* attributes
		//TODO: add ceck for cx-* attributes
		if len(fg)-1 == len(openTeg) {
			continue
		}
		if len(fg) == 0 || err != nil {
			break
		}
		sPos = str.cnt - len(openTeg)
		ePos = sPos + len(fg)
		fmt.Printf("Read fragment: %q\nparameters: len: %d, cutFrom: %d, cutTill: %d\nerr: %v\n", fg, len(fg), sPos, ePos, err)
		//replace the fragment code with placeholder in tpl
		if ePos < len(tpl) {
			tpl = append(
				tpl[:sPos],
				append(
					[]byte(fmt.Sprintf("{{fragment%d}}", fgCnt)),
					tpl[ePos:]...,
				)...)
		} else {
			tpl = append(
				tpl[:sPos],
				[]byte(fmt.Sprintf("{{fragment%d}}", fgCnt))...,
			)
		}
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

func (cx *compoxur) parseTeg(src []byte) *tegCxAttr {
	cx.RLock()
	src = []byte(mustache.Render(string(src), cx.varSet))
	cx.RUnlock()
	endTegPos := bytes.IndexByte(src, '>')
	teg := src[:endTegPos]
	fmt.Printf("Compoxure->parseTeg: %q\n", teg)
	attrSet := bytes.Split(teg, []byte{' '})
	return newTegCxAttr(attrSet)
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
		if pos == -1 {
			continue
		}
		switch string(attr[:pos]) {
		case string(cxUrl):
			tca.cxUrl = attr[(pos + 2):(len(attr) - 1)]
		case string(cxTimeout):
			tca.cxTimeout = attr[(pos + 2):(len(attr) - 1)]
		case string(cxCacheKey):
			tca.cxCacheKey = attr[(pos + 2):(len(attr) - 1)]
		case string(cxCacheTtl):
			tca.cxCacheTtl = attr[(pos + 2):(len(attr) - 1)]
		}
	}
	return tca
}
