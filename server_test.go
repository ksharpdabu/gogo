package gogo

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/dolab/httpdispatch"
	"github.com/dolab/httptesting"
	"github.com/golib/assert"
)

var (
	fakeServer = func() *AppServer {
		logger := NewAppLogger("nil", "")
		logger.SetSkip(3)

		config, _ := fakeConfig("application.json")

		server := NewAppServer(config, logger)

		return server
	}

	fakeHTTPSServer = func() *AppServer {
		logger := NewAppLogger("nil", "")
		logger.SetSkip(3)

		config, _ := fakeConfig("application.https.json")

		server := NewAppServer(config, logger)

		return server
	}
)

func Test_NewAppServer(t *testing.T) {
	it := assert.New(t)

	server := fakeServer()
	it.Implements((*http.Handler)(nil), server)
	it.IsType(&Context{}, server.context.Get())
}

func Test_Server(t *testing.T) {
	server := fakeServer()

	server.GET("/server", func(ctx *Context) {
		ctx.SetStatus(http.StatusNoContent)
		ctx.Return()
	})

	ts := httptesting.NewServer(server, false)
	defer ts.Close()

	request := ts.New(t)
	request.Get("/server", nil)
	request.AssertStatus(http.StatusNoContent)
	request.AssertEmpty()
}

func Benchmark_Server(b *testing.B) {
	it := assert.New(b)
	server := fakeServer()
	server.GET("/server/benchmark", func(ctx *Context) {
		ctx.SetStatus(http.StatusNoContent)
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	r, _ := http.NewRequest("GET", ts.URL+"/server/benchmark", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, r)
	it.Equal(http.StatusNoContent, w.Code)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.ServeHTTP(w, r)
	}
}

func Benchmark_ServerWithReader(b *testing.B) {
	it := assert.New(b)

	reader := []byte("Hello,world!")

	server := fakeServer()
	server.GET("/server/benchmark", func(ctx *Context) {
		ctx.Return(bytes.NewReader(reader))
	})

	r, _ := http.NewRequest("GET", "/server/benchmark", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, r)
	it.Equal(http.StatusOK, w.Code)
	it.Equal(reader, w.Body.Bytes())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.ServeHTTP(w, r)
	}
}

func Test_ServerWithReturn(t *testing.T) {
	server := fakeServer()
	server.GET("/return", func(ctx *Context) {
		if contentType := ctx.Params.Get("content-type"); contentType != "" {
			ctx.SetHeader("Content-Type", contentType)
		}

		data := struct {
			XMLName xml.Name `json:"-"`
			Name    string   `xml:"Name"`
			Age     int      `xml:"Age"`
		}{
			XMLName: xml.Name{
				Local: "Result",
			},
			Name: "gogo",
			Age:  5,
		}

		ctx.Return(data)
	})

	ts := httptesting.NewServer(server, false)
	defer ts.Close()

	// default render
	request := ts.New(t)
	request.Get("/return", nil)
	request.AssertStatus(http.StatusOK)
	request.AssertHeader("Content-Type", "text/plain; charset=utf-8")
	request.AssertContains(`{{ Result} gogo 5}`)

	// json render with request header of accept
	request = ts.New(t)
	request.WithHeader("Accept", "application/json, text/xml, */*; q=0.01")
	request.Get("/return", nil)
	request.AssertStatus(http.StatusOK)
	request.AssertHeader("Content-Type", "application/json")
	request.AssertContains(`{"Name":"gogo","Age":5}`)

	// default render with request query of content-Type=application/json
	params := url.Values{}
	params.Add("content-type", "application/json")

	request = ts.New(t)
	request.Get("/return?"+params.Encode(), nil)
	request.AssertStatus(http.StatusOK)
	request.AssertHeader("Content-Type", "application/json")
	request.AssertContains(`{"Name":"gogo","Age":5}`)

	// xml render with request header of accept
	request = ts.New(t)
	request.WithHeader("Accept", "appication/json, text/xml, */*; q=0.01")
	request.Get("/return", nil)
	request.AssertStatus(http.StatusOK)
	request.AssertHeader("Content-Type", "text/xml")
	request.AssertContains("<Result><Name>gogo</Name><Age>5</Age></Result>")

	// default render with request query of content-Type=text/xml
	params = url.Values{}
	params.Add("content-type", "text/xml")

	request = ts.New(t)
	request.Get("/return?"+params.Encode(), nil)
	request.AssertStatus(http.StatusOK)
	request.AssertHeader("Content-Type", "text/xml")
	request.AssertContains(`<Result><Name>gogo</Name><Age>5</Age></Result>`)
}

func Test_ServerWithNotFound(t *testing.T) {
	server := fakeServer()

	ts := httptest.NewServer(server)
	defer ts.Close()

	request := httptesting.New(ts.URL, false).New(t)
	request.Get("/not/found", nil)
	request.AssertNotFound()
	request.AssertContains("Route(GET /not/found) not found")
}

func Test_ServerWithThroughput(t *testing.T) {
	it := assert.New(t)
	logger := NewAppLogger("nil", "")
	config, _ := fakeConfig("application.throttle.json")

	server := NewAppServer(config, logger)
	server.newThrottle(1)

	server.GET("/server/throughput", func(ctx *Context) {
		ctx.SetStatus(http.StatusNoContent)
		ctx.Return()
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	var (
		wg sync.WaitGroup

		routines = 3
	)

	bufc := make(chan []byte, routines)

	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func() {
			defer wg.Done()

			request := httptesting.New(ts.URL, false).New(t)
			request.Get("/server/throughput", nil)

			bufc <- request.ResponseBody
		}()
	}
	wg.Wait()

	close(bufc)

	s := ""
	for buf := range bufc {
		s += string(buf)
	}

	it.Contains(s, "I'm a teapot")
}

func Test_ServerWithConcurrency(t *testing.T) {
	it := assert.New(t)
	logger := NewAppLogger("nil", "")
	config, _ := fakeConfig("application.throttle.json")

	server := NewAppServer(config, logger)
	server.newSlowdown(1, 1)

	server.GET("/server/concurrency", func(ctx *Context) {
		time.Sleep(time.Second)

		ctx.SetStatus(http.StatusNoContent)
		ctx.Return()
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	var (
		wg sync.WaitGroup

		routines = 3
	)

	bufc := make(chan string, routines)

	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func(routine int) {
			defer wg.Done()

			request := httptesting.New(ts.URL, false).New(t)
			request.Get("/server/concurrency", nil)

			bufc <- fmt.Sprintf("[routine@#%d] %s", routine, string(request.ResponseBody))
		}(i)
	}
	wg.Wait()

	close(bufc)

	s := ""
	for buf := range bufc {
		s += string(buf)
	}

	it.Contains(s, "Too Many Requests")
}

func Test_HTTPS_Server(t *testing.T) {
	server := fakeHTTPSServer()

	server.GET("/https/server", func(ctx *Context) {
		ctx.SetStatus(http.StatusNoContent)
	})

	ts := httptesting.NewServer(server, true)
	defer ts.Close()

	request := ts.New(t)
	request.Get("/https/server", nil)
	request.AssertStatus(http.StatusNoContent)
	request.AssertEmpty()
}

func Test_Server_newContext(t *testing.T) {
	it := assert.New(t)
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "https://www.example.com/resource?key=url_value&test=url_true", nil)
	params := NewAppParams(request, httpdispatch.Params{})

	server := fakeServer()
	ctx := server.newContext(request, "", "", params)
	ctx.run(recorder, nil, nil)

	it.Equal(request, ctx.Request)
	it.Equal(recorder.Header().Get(server.requestID), ctx.Response.Header().Get(server.requestID))
	it.Equal(params, ctx.Params)
	it.Nil(ctx.settings)
	it.Nil(ctx.frozenSettings)
	it.Empty(ctx.middlewares)
	it.EqualValues(math.MaxInt8, ctx.cursor)

	// creation
	newCtx := server.newContext(request, "", "", params)
	newCtx.run(recorder, nil, nil)

	it.NotEqual(fmt.Sprintf("%p", ctx), fmt.Sprintf("%p", newCtx))
}

func Benchmark_Server_newContext(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	server := fakeServer()
	request, _ := http.NewRequest("GET", "https://www.example.com/resource?key=url_value&test=url_true", nil)
	params := NewAppParams(request, httpdispatch.Params{})

	for i := 0; i < b.N; i++ {
		server.newContext(request, "", "", params)
	}
}

func Benchmark_Server_newContextWithReuse(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	server := fakeServer()
	request, _ := http.NewRequest("GET", "https://www.example.com/resource?key=url_value&test=url_true", nil)
	params := NewAppParams(request, httpdispatch.Params{})

	for i := 0; i < b.N; i++ {
		ctx := server.newContext(request, "", "", params)
		server.reuseContext(ctx)
	}
}
