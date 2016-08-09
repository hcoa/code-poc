package main

import (
	"fmt"
	"time"
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
	cackeKey     []byte
	noCache      bool
}

func (b *backend) process(cx *compoxur) []byte {
	//check cache
	ttl := time.ParseDuration(b.ttl)
	res, err := cx.cache.Get(b.cacheKey)
	if err == nil {
		return res
	}

	timeout := time.ParseDuration(b.timeout)
	c := &fasthttp.HostClient{
		Addr:            b.target,
		Name:            cx.getVar("user:agent", ""),
		MaxConnDuration: timeout,
	}

	uri := cx.getVar("url:path", "")
	if len(uri) == 0 {
		//TODO: add error handling here
		return []byte{}
	}

	var statusCode int
	statusCode, res, err = c.GetTimeout(res, uri, timeout)
	if statusCode != fasthttp.StatusOK {
		return []byte(fmt.Sprintf(errTpl, err))
	}
}
