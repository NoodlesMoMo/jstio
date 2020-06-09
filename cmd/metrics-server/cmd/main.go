package main

import (
	"context"
	"fmt"
	metricsV2 "github.com/envoyproxy/go-control-plane/envoy/service/metrics/v2"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var (
	logger = logrus.New()
)

type MetricsServer struct{}

func (ms *MetricsServer) doMetricEntries(name string, metrics *io_prometheus_client.MetricFamily) {
	switch metrics.GetType() {
	case io_prometheus_client.MetricType_COUNTER:
		for _, mc := range metrics.Metric {
			fmt.Printf("### %s counter value: %f\n", name, *mc.Counter.Value)
			for _, label := range mc.Label {
				fmt.Println(">>>> label:", label.Name, " value:", label.Value)
			}
		}
	case io_prometheus_client.MetricType_GAUGE:
		for _, mc := range metrics.Metric {
			fmt.Printf("### %s gauge value: %f\n", name, *mc.Gauge.Value)
			for _, label := range mc.Label {
				fmt.Println(">>>> label:", label.Name, " value:", label.Value)
			}
		}
	case io_prometheus_client.MetricType_SUMMARY:
		for _, mc := range metrics.Metric {
			//fmt.Println("### summary value:", mc.Summary.String())
			for _, label := range mc.Label {
				fmt.Println(">>>> label:", label.Name, " value:", label.Value)
			}
		}
	case io_prometheus_client.MetricType_UNTYPED:
	case io_prometheus_client.MetricType_HISTOGRAM:
		for _, mc := range metrics.Metric {
			//fmt.Println("### histogram value:", mc.Histogram.String())
			for _, label := range mc.Label {
				fmt.Println(">>>> label", label.Name, " value:", label.Value)
			}
		}
	}
}

func (ms *MetricsServer) StreamMetrics(stream metricsV2.MetricsService_StreamMetricsServer) error {
	var err error
	for {
		msg, err := stream.Recv()
		// FIXME: continue or break, WTF!
		if err != nil {
			logger.WithError(err).Errorln("Oops! recv error:", err)
			break
		}

		if msg.Identifier != nil {
			//meta.Pod = msg.Identifier.Node.Id
			//meta.OdinCluster = msg.Identifier.Node.Metadata.Fields["odin_cluster"].GetStringValue()
			//meta.App = msg.Identifier.Node.Cluster
			//meta.Level, meta.FileName = adapter.JstioLevelScan(msg.Identifier.LogName)
			//meta.Domain = meta.App + "." + meta.OdinCluster + ".odin.sogou"
		}

		for _, mc := range msg.EnvoyMetrics {
			logger.WithFields(logrus.Fields{
				"metric_name": mc.GetName(),
				"metric_help": mc.GetHelp(),
			})
			ms.doMetricEntries(mc.GetName(), mc)
		}
	}

	return err
}

func signalAction(cancel context.CancelFunc) {
	sc := make(chan os.Signal)
	signal.Notify(sc)

	for s := range sc {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP:
			cancel()
		case syscall.SIGURG:
		default:
			logger.Errorln("un-catch signal:", s)
		}
	}
}

func main() {
	var (
		listenPort   uint
		region, conf string
	)

	app := cli.NewApp()
	app.Name = "metrics-server"
	app.Usage = "jstio metrics server"
	app.Version = `v0.1.0`

	app.Flags = []cli.Flag{
		cli.UintFlag{
			Name:        "port,p",
			Usage:       "listen port",
			Destination: &listenPort,
			Value:       19991,
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
				return pwd + "/conf/jmss.yaml"
			}(),
		},
	}

	app.Before = func(ctx *cli.Context) error {
		logger.SetOutput(os.Stderr)
		return nil
	}

	app.Action = func(cmdContext *cli.Context) {
		//_, err := options.MustLoadRegionOptions(conf, region)
		//if err != nil {
		//	panic(err)
		//}
		ctx, cancel := context.WithCancel(context.Background())
		go signalAction(cancel)

		server := MetricsServer{}
		RunMetricsServer(ctx, &server, listenPort)
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func RunMetricsServer(ctx context.Context, svc *MetricsServer, port uint) {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	metricsV2.RegisterMetricsServiceServer(grpcServer, svc)

	logger.Println("metrics log server will listen on: ", port)

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
