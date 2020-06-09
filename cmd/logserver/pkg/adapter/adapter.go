package adapter

import (
	"errors"
	"fmt"
	alg "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v2"
	"os"
	"strings"
)

type LogLevel = string

const (
	LogLevelDebug   = `debug`
	LogLevelNon200  = `non200`
	LogLevelTimeout = `timeout`
)

var (
	_loggers = map[string]JstioLogger{}

	_logLevels = map[string]LogLevel{
		"debug":   LogLevelDebug,
		"non200":  LogLevelNon200,
		"timeout": LogLevelTimeout,
	}
)

type MetaData struct {
	App         string `json:"app"`
	Pod         string `json:"pod"`
	OdinCluster string `json:"oc"`
	Domain      string `json:"domain"`
	FileName    string `json:"file_name"`
	Level       string `json:"level"`
}

type JstioLogger interface {
	Sync(meta *MetaData, entry *alg.HTTPAccessLogEntry) error
}

// RegisterAdapter: warning not thread safe!!!
func RegisterAdapter(name string, logger JstioLogger) error {
	_, ok := _loggers[name]
	if ok {
		return errors.New("has existed")
	}

	fmt.Println("adapter:", name, "register")

	_loggers[name] = logger
	return nil
}

func JstioLevelScan(fileName string) (level, name string) {
	seps := strings.SplitN(fileName, "/", 2)
	switch len(seps) {
	case 0:
		return LogLevelDebug, `unknown_name`
	case 1:
		level, name = LogLevelDebug, `unknown_name`
		if lv, ok := _logLevels[level]; ok {
			level = lv
		}
	default:
		level, name = seps[0], seps[1]
		if name == "" {
			name = `unknown_name`
		}
		if _, ok := _logLevels[level]; !ok {
			level = LogLevelDebug
		}
	}
	return
}

func SyncJstioLogs(meta *MetaData, entry *alg.HTTPAccessLogEntry) {
	for name, ada := range _loggers {
		if err := ada.Sync(meta, entry); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "adapter %s, error: %s\n", name, err.Error())
		}
	}
}
