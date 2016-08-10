package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	fragmentChanBufferSize   = 30
	workerFragmentProcessCnt = 5
)

type backend struct {
	regexp []byte
	target string
	host   string
	//e.g. 1s = 1000, 1m = 60*1000 etc.
	//The valid values are 1s, 1m, 1h
	//If you do not provide a suffix it assumes ms
	ttl          []byte
	timeout      []byte
	quietFailure bool
	dontPassUrl  bool
	contentTypes []string
	headers      []string
	cacheKey     []byte
	noCache      bool
	fc           *fasthttp.HostClient
}

func (b *backend) process(cx *compoxur, tca *tegCxAttr) []byte {
	if len(tca.cxTimeout) == 0 {
		tca.cxTimeout = b.timeout
	}
	if len(tca.cxCacheKey) == 0 {
		tca.cxCacheKey = b.cacheKey
	}
	if len(tca.cxCacheTtl) == 0 {
		tca.cxCacheTtl = b.ttl
	}
	//check cache
	res, err := cx.cache.Get(tca.cxCacheKey)
	if err == nil {
		return res
	}
	timeout, _ := time.ParseDuration(string(tca.cxTimeout))
	/*
		c := &fasthttp.HostClient{
			Addr:            b.target,
			Name:            cx.getVar("user:agent", ""),
			MaxConnDuration: timeout,
		}
	*/
	uri := tca.cxUrl
	if len(uri) == 0 {
		return []byte(fmt.Sprintf(errTpl, "URL path is empty!"))
	}
	fmt.Println(b.host, timeout, string(uri))

	var statusCode, fCnt, i int
	statusCode, res, err = b.fc.GetTimeout(res, string(uri), timeout)
	fmt.Printf("response: %q\nstatusCode: %d\n", res, statusCode)
	if statusCode != fasthttp.StatusOK {
		return []byte(fmt.Sprintf(errTpl, err))
	}

	fInCh := make(chan *fragment, fragmentChanBufferSize)
	fOutCh := make(chan *fragment, fragmentChanBufferSize)
	for i = 0; i < workerFragmentProcessCnt; i++ {
		go parse(fInCh, fOutCh, timeout, cx)
	}
	res, fCnt = cx.parseHtml(res, b, fInCh)
	fmt.Printf("tpl: %q\nfragment count: %d\n", res, fCnt)

	key := make([]byte, 0, 20)
	for i = 0; i < fCnt; i++ {
		select {
		case f := <-fOutCh:
			key = []byte(fmt.Sprintf("{{fragment%d}}", f.id))
			res = bytes.Replace(res, key, f.res, 1)
		case <-time.After(timeout):
			//TODO: return result and set error in all unfieled placeholders
			return []byte(fmt.Sprintf(errTpl, "Timeout is reached"))
		}
	}

	//here we have processed html from backend
	//let's save it to cache
	ttl, _ := time.ParseDuration(string(tca.cxCacheTtl))
	ttls := int(ttl / time.Second)
	cx.cache.Set(tca.cxCacheKey, res, ttls)

	return res
}
