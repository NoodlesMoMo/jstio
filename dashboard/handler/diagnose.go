package handler

import (
	routing "github.com/qiangxue/fasthttp-routing"
	"jstio/dashboard/service"
)

func DiagnoseHandler(ctx *routing.Context) error {

	_ = service.DiagnoseService(ctx)

	return nil
}
