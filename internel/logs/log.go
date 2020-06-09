package logs

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
)

var (
	Logger *JstioLogger
	once_  sync.Once
)

type JstioLogger struct {
	*log.Logger
}

func (l *JstioLogger) Output(dep int, s string) error {
	l.WithField(`[MQ]`, `NSQ`).Warningln(s)
	return nil
}

func (l *JstioLogger) Print(v ...interface{}) {
	l.WithField(`[DB]`, `MYSQL`).Println(v...)
}

func MustInitialization(basePath string) {
	if Logger != nil {
		return
	}

	if basePath == "" {
		currDir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		basePath = currDir
	}

	logName := path.Join(basePath, "logs/jstio.%Y-%m-%d_%H") // sogou hadoop log name style
	linkName := path.Join(basePath, "jstio.log")

	once_.Do(func() {
		rl, err := rotatelogs.New(
			logName,
			rotatelogs.WithRotationTime(time.Hour),
			rotatelogs.WithLinkName(linkName),
		)
		if err != nil {
			panic(err)
		}
		Logger = &JstioLogger{log.New()}
		Logger.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
		Logger.SetOutput(rl)
		Logger.SetLevel(log.DebugLevel)
	})
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
