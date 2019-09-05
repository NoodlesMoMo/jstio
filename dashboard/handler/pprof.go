package handler

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"time"

	"github.com/qiangxue/fasthttp-routing"
)

type handler string

func CommandLine(ctx *routing.Context) error {
	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	ctx.SetContentType("text/plain; charset=utf-8")
	_, err := fmt.Fprintf(ctx.Response.BodyWriter(), strings.Join(os.Args, "\x00"))

	return err
}

func Profile(ctx *routing.Context) error {

	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	sec, _ := strconv.ParseInt(string(ctx.QueryArgs().Peek("seconds")), 10, 64)
	if sec == 0 {
		sec = 30
	}

	ctx.SetContentType("application/octet-stream")
	ctx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="profile"`))
	if err := pprof.StartCPUProfile(ctx.Response.BodyWriter()); err != nil {
		_, _ = ctx.WriteString(fmt.Sprintf("Could not enable CPU profiling: %s", err))
		return err
	}

	time.Sleep(time.Duration(sec * int64(time.Second)))

	pprof.StopCPUProfile()

	return nil
}

func TraceX(ctx *routing.Context) error {
	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	sec, err := strconv.ParseFloat(string(ctx.QueryArgs().Peek("seconds")), 61)
	if sec <= 0 || err != nil {
		sec = 1
	}

	ctx.SetContentType("application/octet-stream")
	ctx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="profile"`))

	if err := trace.Start(ctx.Response.BodyWriter()); err != nil {
		_, _ = ctx.WriteString(fmt.Sprintf("Could not enable tracing: %s", err.Error()))
		return err
	}

	time.Sleep(time.Duration(sec * float64(time.Second)))
	trace.Stop()

	return nil
}

func Symbol(ctx *routing.Context) error {
	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	ctx.SetContentType("text/plain; charset=utf-8")

	// We have to read the whole POST body before
	// writing any output. Buffer the output here.
	var buf bytes.Buffer

	// We don't know how many symbols we have, but we
	// do have symbol information. Pprof only cares whether
	// this number is 0 (no symbols available) or > 0.
	_, _ = fmt.Fprintf(&buf, "num_symbols: 1\n")

	var b *bufio.Reader
	if string(ctx.Method()) == "POST" {
		b = bufio.NewReader(bytes.NewReader(ctx.Request.Body()))
	} else {
		b = bufio.NewReader(bytes.NewReader(ctx.RequestURI())) // FIXME
	}

	for {
		word, err := b.ReadSlice('+')
		if err == nil {
			word = word[0 : len(word)-1] // trim +
		}
		pc, _ := strconv.ParseUint(string(word), 0, 64)
		if pc != 0 {
			f := runtime.FuncForPC(uintptr(pc))
			if f != nil {
				_, _ = fmt.Fprintf(&buf, "%#x %s\n", pc, f.Name())
			}
		}

		// Wait until here to check for err; the last
		// symbol will have an err because it doesn't end in +.
		if err != nil {
			if err != io.EOF {
				_, _ = fmt.Fprintf(&buf, "reading request: %v\n", err)
			}
			break
		}
	}

	_, err := ctx.Write(buf.Bytes())

	return err
}

func (name handler) WTF(ctx *routing.Context) error {
	ctx.Request.Header.Set("X-Content-Type-Options", "nosniff")
	p := pprof.Lookup(string(name))
	if p == nil {
		ctx.Response.SetStatusCode(http.StatusNotFound)
		return nil
	}

	gc, _ := strconv.Atoi(string(ctx.PostArgs().Peek("gc")))
	if name == `heap` && gc > 0 {
		runtime.GC()
	}

	debug, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("debug")))
	if debug != 0 {
		ctx.SetContentType("text/plain; charset=utf-8")
	} else {
		ctx.SetContentType("application/octet-stream")
		ctx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	}

	return p.WriteTo(ctx.Response.BodyWriter(), debug)
}

var indexTmpl = template.Must(template.New("index").Parse(`<html>
<head>
<title>/debug/pprof/</title>
</head>
<body>
/debug/pprof/<br>
<br>
profiles:<br>
<table>
{{range .}}
<tr><td align=right>{{.Count}}<td><a href="{{.Name}}?debug=1">{{.Name}}</a>
{{end}}
</table>
<br>
<a href="goroutine?debug=2">full goroutine stack dump</a><br>
</body>
</html>
`))

func ProfileIndex(ctx *routing.Context) error {
	ctx.SetContentType("text/html; charset=utf-8")
	name := ctx.Param("name")
	if name != "" {
		_ = handler(name).WTF(ctx)
	}

	profiles := pprof.Profiles()
	return indexTmpl.Execute(ctx.Response.BodyWriter(), profiles)
}
