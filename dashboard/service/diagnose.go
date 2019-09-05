package service

import (
	routing "github.com/qiangxue/fasthttp-routing"
	"jstio/model"
)

func DiagnoseService(ctx *routing.Context) error {
	apps, err := model.CompleteActiveApps()
	if err != nil {
		return ErrorResponse(ctx, -1, err)
	}

	return SuccessResponse(ctx, apps)
}
