package main

import "time"

var (
	tegAttrSet = [...]string{
		"cx-url",
		"cx-cacke-key",
		"cx-cacke-ttl",
		"cx-timeout",
		"cx-no-cache",
		"cx-replace-outer",
		"cx-ignore-404",
		"cx-ignore-error",
	}
	paramNameSet = [...]string{
		//Parameters matched from the parameters configuration (regex + name)
		//pairs in the configuration
		///resource/{{param:resourceId}}
		"param",
		//Parameters matched from any query string key values in the incoming URL
		///user/{{query:userId}}
		"query",
		//Any elements of the incoming url (search, query, pathname, path, href)
		///search{{url:search}}
		"url",
		//Any cookie value
		///user/{{cookie:TSL_UserID}}
		"cookie",
		//Any incoming header value
		///user/feature/{{header:x-feature-enabled}}
		"header",
		//A server short name from the configuration in the parameters
		//section of config
		//{{server:feature}}/feature
		"server",
		"env",
		"cdn",
		"user",
		"device",
	}
	errTpl = `<div style="color: red; font-weight: bold; font-family: monospace;">Error: %s </div>`
)

type fragment struct {
	alias   string // use as TPL key
	ttl     time.Duration
	timeout time.Duration
	src     []byte
	res     []byte
}
