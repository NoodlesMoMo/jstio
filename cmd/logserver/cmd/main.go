package main

import (
	"context"
	"encoding/json"
	"fmt"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/adapters/es"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/options"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/pkg/adapter"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	alf "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v2"
	als "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"
	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	logger = logrus.New()

	dummySocketAddress = &envoy_api_v2_core.Address{
		Address: &envoy_api_v2_core.Address_SocketAddress{
			SocketAddress: &envoy_api_v2_core.SocketAddress{Address: "0.0.0.0"},
		},
	}
	dummyZeroWrapperValue = &wrappers.UInt32Value{Value: 0}
	dummyDuration         = &duration.Duration{}
	dummyResponseFlags    = &alf.ResponseFlags{}
	dummyRequestMethod    = envoy_api_v2_core.RequestMethod_METHOD_UNSPECIFIED
)

type AccessLogService struct{}

func (svc *AccessLogService) makeDummyLogEntryCommonProperty() *alf.AccessLogCommon {
	unixTimestamp := time.Now().Unix()
	now := &timestamp.Timestamp{
		Seconds: unixTimestamp,
	}
	dummy := &alf.AccessLogCommon{
		DownstreamRemoteAddress:        dummySocketAddress,
		DownstreamLocalAddress:         dummySocketAddress,
		StartTime:                      now,
		TimeToLastRxByte:               dummyDuration,
		TimeToFirstUpstreamTxByte:      dummyDuration,
		TimeToLastUpstreamTxByte:       dummyDuration,
		TimeToFirstUpstreamRxByte:      dummyDuration,
		TimeToLastUpstreamRxByte:       dummyDuration,
		TimeToFirstDownstreamTxByte:    dummyDuration,
		TimeToLastDownstreamTxByte:     dummyDuration,
		UpstreamRemoteAddress:          dummySocketAddress,
		UpstreamLocalAddress:           dummySocketAddress,
		UpstreamCluster:                "dummy-cluster",
		ResponseFlags:                  dummyResponseFlags,
		Metadata:                       nil,
		UpstreamTransportFailureReason: "",
		RouteName:                      "",
		DownstreamDirectRemoteAddress:  dummySocketAddress,
	}

	return dummy
}

func (svc *AccessLogService) makeDummyRequest() *alf.HTTPRequestProperties {
	dummy := &alf.HTTPRequestProperties{
		RequestMethod: dummyRequestMethod,
		Scheme:        "-",
		Authority:     "dummy",
		Path:          "",
		Port:          dummyZeroWrapperValue,
	}
	return dummy
}

func (svc *AccessLogService) makeDummyResponse() *alf.HTTPResponseProperties {
	dummy := &alf.HTTPResponseProperties{
		ResponseCode: dummyZeroWrapperValue,
	}
	return dummy
}

// StreamAccessLogs implements the access log service.
func (svc *AccessLogService) StreamAccessLogs(stream als.AccessLogService_StreamAccessLogsServer) error {
	var meta = adapter.MetaData{}

	for {
		msg, err := stream.Recv()
		// FIXME: continue or break, WTF!
		if err != nil {
			logger.WithError(err).Errorln("Oops! recv error:", err)
			return err
		}

		if msg.Identifier != nil {
			meta.Pod = msg.Identifier.Node.Id
			meta.OdinCluster = msg.Identifier.Node.Metadata.Fields["odin_cluster"].GetStringValue()
			meta.App = msg.Identifier.Node.Cluster
			meta.Level, meta.FileName = adapter.JstioLevelScan(msg.Identifier.LogName)
			meta.Domain = meta.App + "." + meta.OdinCluster + ".odin.sogou"
		}

		switch entries := msg.LogEntries.(type) {
		case *als.StreamAccessLogsMessage_HttpLogs:
			for _, entry := range entries.HttpLogs.LogEntry {
				if entry == nil {
					logger.Errorln("log entry is null")
					continue
				}
				if entry.CommonProperties == nil {
					entry.CommonProperties = svc.makeDummyLogEntryCommonProperty()
				} else {
					if entry.CommonProperties.UpstreamRemoteAddress == nil {
						entry.CommonProperties.UpstreamRemoteAddress = dummySocketAddress
					}
					if entry.CommonProperties.StartTime == nil {
						unixTimestamp := time.Now().Unix()
						entry.CommonProperties.StartTime = &timestamp.Timestamp{
							Seconds: unixTimestamp,
						}
					}
					if entry.CommonProperties.TimeToLastUpstreamRxByte == nil {
						entry.CommonProperties.TimeToLastUpstreamRxByte = dummyDuration
					}
				}

				if entry.Request == nil {
					entry.Request = svc.makeDummyRequest()
				}
				if entry.Response == nil {
					entry.Response = svc.makeDummyResponse()
				} else {
					if entry.Response.ResponseCode == nil {
						entry.Response.ResponseCode = dummyZeroWrapperValue
					}
				}
				adapter.SyncJstioLogs(&meta, entry)
			}
		case *als.StreamAccessLogsMessage_TcpLogs:
			for _, entry := range entries.TcpLogs.LogEntry {
				if entry != nil {
					// TODO: add later ...
					//common := entry.CommonProperties
					//if common == nil {
					//	common = &alf.AccessLogCommon{}
					//}
				}
			}
		}
	}
}

func RunAccessLogServer(ctx context.Context, svc *AccessLogService, port uint) {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	als.RegisterAccessLogServiceServer(grpcServer, svc)
	logger.Println("access log server will listen on: ", port)

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Println(err)
		}
	}()
	<-ctx.Done()

	//FIXME:
	//grpcServer.GracefulStop()
	grpcServer.Stop()
}

func RunHTTPServer(port uint) {
	r := routing.New()

	r.Get("/es/stat", ElasticAggSearchHandler)

	server := fasthttp.Server{
		ReadTimeout: 5 * time.Second,
	}
	server.Handler = r.HandleRequest

	logger.Println("http server will listen on: ", port)
	if err := server.ListenAndServe(fmt.Sprintf(":%d", port)); err != nil {
		panic(err)
	}
}

func ElasticAggSearchHandler(ctx *routing.Context) error {

	const timeFormat = `2006-01-02 15:04:05`

	fs, ts := string(ctx.QueryArgs().Peek("from")), string(ctx.QueryArgs().Peek("to"))
	now := time.Now()

	defaultFrom := func() time.Time {
		d, _ := time.ParseDuration("-5m")
		return now.Add(d)
	}

	from, to := defaultFrom(), now

	if fs != "" {
		if t, err := time.ParseInLocation(timeFormat, fs, time.Local); err == nil {
			from = t
		}
	}

	if ts != "" {
		if t, err := time.ParseInLocation(timeFormat, ts, time.Local); err == nil {
			to = t
		}
	}

	result, err := es.GetElasticAdapter().Search(from, to)
	if err != nil {
		logger.WithError(err).Errorln("elastic error: ", err)
	}

	b, _ := json.Marshal(result)
	_, _ = ctx.Write(b)

	return nil
}

func signalAction(cancel context.CancelFunc) {
	sc := make(chan os.Signal)
	signal.Notify(sc)

	for s := range sc {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP:
			cancel()
		default:
			logger.Errorln("un-catch signal:", s)
		}
	}
}

func main() {

	var (
		listenPort, httpListenPort uint
		region, conf               string
	)

	app := cli.NewApp()
	app.Name = "logserver"
	app.Usage = "jstio logserver"
	app.Version = `v0.1.0`

	app.Flags = []cli.Flag{
		cli.UintFlag{
			Name:        "port,p",
			Usage:       "listen port",
			Destination: &listenPort,
			Value:       19981,
		},
		cli.UintFlag{
			Name:        "http_port",
			Usage:       "http listen port",
			Destination: &httpListenPort,
			Value:       19982,
		},
		cli.StringFlag{
			Name:        "region,r",
			Usage:       "run region",
			Destination: &region,
			Value:       "develop",
		},
		cli.StringFlag{
			Name:        "config,c",
			Usage:       "config file",
			Destination: &conf,
			Value: func() string {
				pwd, _ := os.Getwd()
				return pwd + "/conf/logserver.yaml"
			}(),
		},
	}

	app.Before = func(ctx *cli.Context) error {
		logger.SetOutput(os.Stderr)
		return nil
	}

	app.Action = func(cmdContext *cli.Context) {
		_, err := options.MustLoadRegionOptions(conf, region)
		if err != nil {
			panic(err)
		}

		//_ = adapter.RegisterAdapter(`native-disk`, disk.GetDiskAngLogAdapter())
		_ = adapter.RegisterAdapter(`elastic-search`, es.GetElasticAdapter())

		ctx, cancel := context.WithCancel(context.Background())
		go signalAction(cancel)
		go RunHTTPServer(httpListenPort)

		server := AccessLogService{}
		RunAccessLogServer(ctx, &server, listenPort)
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
