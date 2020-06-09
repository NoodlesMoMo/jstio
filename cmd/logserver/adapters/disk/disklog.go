package disk

import (
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/options"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"git.sogou-inc.com/iweb/jstio/cmd/logserver/pkg/adapter"

	"git.sogou-inc.com/iweb/jstio/cmd/logserver/pkg/diskany"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	alf "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v2"
)

var (
	_diskLogger *NativeDiskLogger
	_diskOnce   = sync.Once{}

	//diskLogger = NativeDiskLogger{
	//	diskLogger: diskany.GetDiskLogger(),
	//}
)

type NativeDiskLogger struct {
	diskLogger *diskany.DiskLogServer
}

//func init() {
//	if err := adapter.RegisterAdapter(`native-disk`, &diskLogger); err != nil {
//		panic(err)
//	}
//}

func rotatedLogName(logPath, logName string) string {
	return logPath + time.Now().Format("/20060102/"+logName+".15.log")
}

func httpLogFormat(meta *adapter.MetaData, entry *alf.HTTPAccessLogEntry) []byte {
	req, resp, properties := entry.Request, entry.Response, entry.CommonProperties
	line := strings.Join([]string{
		meta.Pod,
		properties.StartTime.String(),

		req.RequestId,
		req.Scheme,
		req.RequestMethod.String(),
		req.ForwardedFor,
		req.Authority,
		req.Path,
		strconv.Itoa(int(req.RequestHeadersBytes)),
		strconv.Itoa(int(req.RequestBodyBytes)),

		properties.ResponseFlags.String(),
		//properties.TimeToLastRxByte.String(),
		properties.TimeToLastUpstreamRxByte.String(),
		properties.UpstreamCluster,
		properties.UpstreamRemoteAddress.Address.(*envoy_api_v2_core.Address_SocketAddress).SocketAddress.Address,
		properties.UpstreamTransportFailureReason,

		strconv.Itoa(int(resp.ResponseCode.Value)),
		strconv.Itoa(int(resp.ResponseHeadersBytes)),
		strconv.Itoa(int(resp.ResponseBodyBytes)),
	}, "|")

	return *(*[]byte)(unsafe.Pointer(&line))
}

func (d *NativeDiskLogger) Sync(meta *adapter.MetaData, entry *alf.HTTPAccessLogEntry) error {
	logPath := options.GetOption().DiskLogRoot + meta.Domain
	logName := rotatedLogName(logPath, meta.FileName)
	return d.diskLogger.WriteBytes(logName, httpLogFormat(meta, entry))
}

func GetDiskAngLogAdapter() *NativeDiskLogger {
	if _diskLogger != nil {
		return _diskLogger
	}

	_diskOnce.Do(func() {
		_diskLogger = &NativeDiskLogger{
			diskLogger: diskany.GetDiskLogger(),
		}
	})

	return _diskLogger
}
