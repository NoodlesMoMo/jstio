package handler

import (
	"git.sogou-inc.com/iweb/jstio/dashboard/service"
	"git.sogou-inc.com/iweb/jstio/model"
	routing "github.com/qiangxue/fasthttp-routing"
)

func DiagnoseAppsHandler(ctx *routing.Context) error {

	_ = service.DiagnoseService(ctx)

	return nil
}

func DiagnosePodWaitHandler(ctx *routing.Context) error {

	data := model.PodWaitDump()

	_, _ = ctx.Write(data)

	return nil
}
