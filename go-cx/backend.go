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
	regexp string
	target string
	host   string
	//e.g. 1s = 1000, 1m = 60*1000 etc.
	//The valid values are 1s, 1m, 1h
	//If you do not provide a suffix it assumes ms
	ttl          string
	timeout      string
	quietFailure bool
	dontPassUrl  bool
	contentTypes []string
	headers      []string
	cacheKey     []byte
	noCache      bool
	fc           *fasthttp.HostClient
}

func (b *backend) process(cx *compoxur) []byte {
	//check cache
	res, err := cx.cache.Get(b.cacheKey)
	if err == nil {
		return res
	}
	timeout, _ := time.ParseDuration(b.timeout)
	/*
		c := &fasthttp.HostClient{
			Addr:            b.target,
			Name:            cx.getVar("user:agent", ""),
			MaxConnDuration: timeout,
		}
	*/
	uri := cx.getVar("url:path", "")
	if len(uri) == 0 {
		return []byte(fmt.Sprintf(errTpl, "URL path is empty!"))
	}
	fmt.Println(b.host, timeout, uri)

	var statusCode, fCnt, i int
	statusCode, res, err = b.fc.GetTimeout(res, uri, timeout)
	fmt.Printf("response: %q\nstatusCode: %d", res, statusCode)
	if statusCode != fasthttp.StatusOK {
		return []byte(fmt.Sprintf(errTpl, err))
	}

	fInCh := make(chan *fragment, fragmentChanBufferSize)
	fOutCh := make(chan *fragment, fragmentChanBufferSize)
	for i = 0; i < workerFragmentProcessCnt; i++ {
		go parse(fInCh, fOutCh, timeout)
	}
	res, fCnt = cx.parseHtml(res, b, fInCh)
	fmt.Printf("tpl: %q\nfragment count: %d", res, fCnt)

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
	ttl, _ := time.ParseDuration(b.ttl)
	ttls := int(ttl / time.Second)
	cx.cache.Set(b.cacheKey, res, ttls)

	return res
}
