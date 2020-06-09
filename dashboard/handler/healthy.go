package handler

import (
	"git.sogou-inc.com/iweb/jstio/internel/logs"
	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/sirupsen/logrus"
)

func HealthyHandler(ctx *routing.Context) error {
	app := string(ctx.Request.Header.Peek("app_id"))
	logs.Logger.WithFields(logrus.Fields{
		"app":      app,
		"response": "healthy",
	}).Println()

	return nil
}
