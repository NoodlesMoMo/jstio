package handler

import (
	. "jstio/internel/logs"
	"jstio/model"
	"runtime/debug"

	"github.com/qiangxue/fasthttp-routing"
	"github.com/sirupsen/logrus"
)

func AccessLog(ctx *routing.Context) error {
	Logger.WithFields(logrus.Fields{
		"path":   string(ctx.Path()),
		"method": string(ctx.Method()),
		"query":  string(ctx.Request.RequestURI()),
		"body":   string(ctx.Request.Body()),
	}).Println()

	return ctx.Next()
}

func Auth(ctx *routing.Context) error {
	ctx.Set(`authority`, &model.UserData{
		UserId:   `12345`,
		UserName: `jack`,
		Email:    `jack@sogou-inc.com`,
	})

	return ctx.Next()
}

func ExceptionDump(ctx *routing.Context) error {
	defer func(ctx *routing.Context) {
		if r := recover(); r != nil {
			Logger.WithField("exception recover", string(ctx.RequestURI())).Errorln(r)
			Logger.Errorln(string(debug.Stack()))
		}
	}(ctx)

	return ctx.Next()
}
