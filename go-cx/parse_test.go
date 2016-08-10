package main

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/coocood/freecache"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

const (
	comp1 = `<div>
  <p>Widget 1</p>
</div>`
	comp2 = `<div>
  <p>Widget Two</p>
  <div cx-url='{{server:local}}/application/component1'></div>
</div>`
	root = `<body>
  <div cx-url='{{server:local}}/application/component2'></div>
</body>`
	compoxured = `<body>
  <div cx-url='{{server:local}}/application/component2'>
    <div>
      <p>Widget Two</p>
      <div cx-url='{{server:local}}/application/component1'>
        <div>
          <p>Widget 1</p>
        </div>
      </div>
    </div>
  </div>
</body>`
)

var respBodySet = map[string]string{
	"comp1": comp1,
	"comp2": comp2,
	"root":  root,
}

func NewTestServer(t *testing.T) (*fasthttp.Server, *fasthttp.HostClient, chan struct{}) {
	ln := fasthttputil.NewInmemoryListener()

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Path()) {
			case "/comp1":
				ctx.WriteString(comp1)
			case "/comp2":
				ctx.WriteString(comp2)
			case "/root":
				ctx.WriteString(root)
			default:
				ctx.Success("text/plain", ctx.Path())
			}
		},
	}

	serverStopCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		close(serverStopCh)
	}()

	c := &fasthttp.HostClient{
		Addr: "foobar",
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
		MaxConnDuration: 10 * time.Millisecond,
	}
	return s, c, serverStopCh
}

func NewTestCompoxur(t *testing.T, fc *fasthttp.HostClient) *compoxur {
	cacheSize := 100 * 1024 * 1024
	c := freecache.NewCache(cacheSize)

	varSet := make(map[string]string)
	varSet["server:local"] = "foobar"

	return &compoxur{
		cache:  c,
		varSet: varSet,
		backSet: []backend{
			backend{
				regexp:       "*",
				target:       "http://foobar",
				host:         "test",
				ttl:          "10s",
				timeout:      "1s",
				quietFailure: true,
				dontPassUrl:  false,
				contentTypes: []string{
					"html",
				},
				cacheKey: []byte(`test`),
				fc:       fc,
			},
		},
	}
}

func TestCalls(t *testing.T) {
	_, c, _ := NewTestServer(t)

	for _, alias := range [3]string{"comp1", "comp2", "root"} {
		uri := fmt.Sprintf("http://foobar/%s", alias)
		helperGetTimeout(c, uri, respBodySet[alias], t)
	}
}

func TestCompoxing(t *testing.T) {
	_, c, _ := NewTestServer(t)
	cx := NewTestCompoxur(t, c)
	cx.setVar("url:path", "http://foobar/root")
	res := cx.processBack("root")
	if string(res) != compoxured {
		t.Errorf("got: %q\nexpect: %s\n", res, compoxured)
	}
}

func helperGetTimeout(c *fasthttp.HostClient, uri, bodyStr string, t *testing.T) {
	for i := 0; i < 9; i++ {
		statusCode, body, err := c.GetTimeout(nil, uri, time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if statusCode != fasthttp.StatusOK {
			t.Fatalf("unexpected status code %d. Expecting %d", statusCode, fasthttp.StatusOK)
		}
		if string(body) != bodyStr {
			t.Fatalf("unexpected body %q. Expecting %q", body, bodyStr)
		}
	}
}

func (cx *compoxur) setVar(key, val string) {
	cx.Lock()
	defer cx.Unlock()
	cx.varSet[key] = val
}
