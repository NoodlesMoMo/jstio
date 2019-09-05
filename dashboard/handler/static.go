package handler

import (
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
	"jstio/dashboard/service"
	"jstio/internel"
)

var (
	fsHandler_ = fasthttp.FSHandler(service.TemplateDir("/"), 0)
)

func StaticHandler(ctx *routing.Context) error {
	if internel.GetAfxMeta().DebugMode {
		ctx.Response.Header.Set("Cache-Control", "no-cache")
	}

	fsHandler_(ctx.RequestCtx)

	return nil
}
