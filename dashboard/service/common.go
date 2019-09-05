package service

import (
	"encoding/json"
	"fmt"
	. "jstio/internel/logs"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/qiangxue/fasthttp-routing"
)

type RestResult struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func NotFound(ctx *routing.Context) error {
	return RenderTemplate(ctx, `404.html`, nil)
}

func pathFormat(org string) string {
	name := make([]byte, 0)

	if strings.HasPrefix(org, `HTML`) {
		name = append(name, []byte("HTML_")...)
		org = org[4:]
	}

	for idx, c := range org {
		if byte(c) >= 'A' && byte(c) <= 'Z' && idx != 0 {
			name = append(name, '_')
		}
		name = append(name, byte(c))
	}

	return string(name)
}

func ParamUnpack(ctx *routing.Context, out interface{}) error {

	var err error

	v := reflect.ValueOf(out).Elem()

	for i := 0; i < v.NumField(); i++ {
		fieldInfo := v.Type().Field(i)
		jsonTag := fieldInfo.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		vv := v.Field(i)
		if vv.Kind() == reflect.Slice {
			// FIXME: only fetch data from body ...
			params, err := url.ParseQuery(string(ctx.PostBody()))
			if err != nil || params == nil {
				continue
			}

			param := []string(params[jsonTag])
			if len(param) == 0 {
				continue
			}

			for _, pv := range param {
				elem := reflect.New(vv.Type().Elem()).Elem()
				if e := baseTypeConvert(elem, pv); e != nil {
					err = e
				}

				vv.Set(reflect.Append(vv, elem))
			}

			continue
		}

		pv := string(ctx.FormValue(jsonTag))
		if e := baseTypeConvert(v.Field(i), pv); e != nil {
			err = e
		}
	}

	return err
}

func baseTypeConvert(in reflect.Value, value string) error {
	switch in.Kind() {
	case reflect.String:
		in.SetString(value)
	case reflect.Int:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		in.SetInt(i)
	case reflect.Uint:
		i, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		in.SetUint(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		in.SetBool(b)
	default:
		return fmt.Errorf("unsupported kind %s", in.Type())
	}

	return nil
}

func SuccessResponse(ctx *routing.Context, data interface{}) error {
	ctx.Serialize = json.Marshal

	return ctx.WriteData(RestResult{Data: data})
}

func ErrorResponse(ctx *routing.Context, code int, err interface{}) error {
	ctx.Serialize = json.Marshal

	var msg string
	switch err.(type) {
	case string:
		msg = err.(string)
	case error:
		msg = err.(error).Error()
	default:
		msg = "Oops: what the fuck!"
	}

	return ctx.WriteData(RestResult{Code: code, Message: msg})
}

func ErrorLog(key, val string, err error) {
	if err != nil {
		Logger.WithField(key, val).Errorln(err)
	}
}
