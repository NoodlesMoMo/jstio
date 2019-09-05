package logs

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

var Logger = log.New()

func init() {
	Logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	Logger.SetOutput(os.Stdout)
	Logger.SetLevel(log.DebugLevel)
}

func TaggedLoggerFactory(key string) func(string) *log.Entry {
	return func(tag string) *log.Entry {
		return Logger.WithField(key, tag)
	}
}

func FuncTaggedLoggerFactory() func(string) *log.Entry {
	var key string

	pc, _, _, ok := runtime.Caller(1)
	if ok {
		fn := runtime.FuncForPC(pc)
		_, line := fn.FileLine(pc)

		fnName := fn.Name()
		idx := strings.LastIndexByte(fnName, '.')
		if idx >= 0 {
			fnName = fnName[idx+1:]
		}

		key = fmt.Sprintf("[%s:%d]", fnName, line)
	} else {
		key = `[unknown_caller]`
	}

	return func(tag string) *log.Entry {
		return Logger.WithField(key, tag)
	}
}

type NSQLogger struct{}

func (nl NSQLogger) Output(dep int, s string) error {
	Logger.WithField(`[NSQ]`, `consume`).Warningln(s)
	return nil
}
