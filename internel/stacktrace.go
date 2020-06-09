package internel

import (
	"git.sogou-inc.com/iweb/jstio/internel/logs"
	"runtime/debug"
)

func CoreDump() {
	if r := recover(); r != nil {
		logs.Logger.Println("============= core dump =============")
		if err, ok := r.(error); ok {
			logs.Logger.Errorln(">>> panic error:", err)
		}
		logs.Logger.Fatalln(string(debug.Stack()))
	}
}
