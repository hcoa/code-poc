package main

import (
	"flag"
	"log"
	"sync"

	"github.com/coocood/freecache"
	"github.com/valyala/fasthttp"
)

var (
	addr   = flag.String("addr", ":8080", "TCP address to listen to")
	slocal = flag.String("slocal", "http://localhost:8081", "TCP address as parameters")
)

func main() {
	flag.Parse()

	cacheSize := 100 * 1024 * 1024
	c := freecache.NewCache(cacheSize)

	cnf = make(map[string]string)
	cnf["server:local"] = slocal

	ch := &cxHandler{
		cache:    c,
		paramSet: make(map[string]string),
	}

	if err := fasthttp.ListenAndServe(*addr, ch.processHandle); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

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

type cxHandler struct {
	sync.Mutex
	cache    *freecache.Cache
	paramSet map[string]string
	backSet  []backend
}

func (cx *cxHandler) processHandle(ctx *fasthttp.RequestCtx) {

}

/*
func proxyHandler(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "Hello, world!\n\n")

	fmt.Fprintf(ctx, "Request method is %q\n", ctx.Method())
	fmt.Fprintf(ctx, "RequestURI is %q\n", ctx.RequestURI())
	fmt.Fprintf(ctx, "Requested path is %q\n", ctx.Path())
	fmt.Fprintf(ctx, "Host is %q\n", ctx.Host())
	fmt.Fprintf(ctx, "Query string is %q\n", ctx.QueryArgs())
	fmt.Fprintf(ctx, "User-Agent is %q\n", ctx.UserAgent())
	fmt.Fprintf(ctx, "Connection has been established at %s\n", ctx.ConnTime())
	fmt.Fprintf(ctx, "Request has been started at %s\n", ctx.Time())
	fmt.Fprintf(ctx, "Serial request number for the current connection is %d\n", ctx.ConnRequestNum())
	fmt.Fprintf(ctx, "Your ip is %q\n\n", ctx.RemoteIP())

	fmt.Fprintf(ctx, "Raw request is:\n---CUT---\n%s\n---CUT---", &ctx.Request)

	ctx.SetContentType("text/plain; charset=utf8")

	// Set arbitrary headers
	ctx.Response.Header.Set("X-My-Header", "my-header-value")

	// Set cookies
	var c fasthttp.Cookie
	c.SetKey("cookie-name")
	c.SetValue("cookie-value")
	ctx.Response.Header.SetCookie(&c)

}
*/
