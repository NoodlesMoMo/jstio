package handler

import (
	"git.sogou-inc.com/iweb/jstio/dashboard/service"
	"git.sogou-inc.com/iweb/jstio/internel"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
)

var (
	fsHandler_ = fasthttp.FSHandler(service.TemplateDir("/"), 0)
)

func StaticHandler(ctx *routing.Context) error {
	if internel.GetAfxOption().DebugMode {
		ctx.Response.Header.Set("Cache-Control", "no-cache")
	}

	fsHandler_(ctx.RequestCtx)

	return nil
}
