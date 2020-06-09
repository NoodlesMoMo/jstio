package compass

import (
	"context"
	"git.sogou-inc.com/iweb/jstio/compass/callback"
	"git.sogou-inc.com/iweb/jstio/dashboard"
	"git.sogou-inc.com/iweb/jstio/dashboard/service"
	"git.sogou-inc.com/iweb/jstio/internel"
	"git.sogou-inc.com/iweb/jstio/internel/logs"
	"git.sogou-inc.com/iweb/jstio/model"
	"net"
	"net/http"
	"time"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xcore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	xdiscovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
)

type noCopy struct{}

func (*noCopy) Lock() {}

type Compass struct {
	noCopy
	*xcore.Node

	cache *model.XdsCache

	options *internel.RegionOptions
	stopper *internel.GracefulStopper
	done    context.Context
	cancel  context.CancelFunc
}

func NewCompass() *Compass {

	compass := &Compass{
		stopper: internel.NewGracefulStopper(),
	}

	if err := compass.initialization(); err != nil {
		logs.Logger.Panic(err)
	}

	return compass
}

func (c *Compass) initialization() error {
	var err error

	c.done, c.cancel = context.WithCancel(context.Background())

	// load metadata from config file
	c.options = internel.GetAfxOption()

	c.autoMigrateTables(c.options.DebugMode)

	c.cache = model.MustNewXdsCache(c.done)

	c.stopper.RegistryExitHook(`internal done control`, func() error {
		c.cancel()
		time.Sleep(300 * time.Millisecond)
		return nil
	})

	return err
}

func (c *Compass) autoMigrateTables(auto bool) {
	if auto {
		model.MigrateAppsTables()
		model.MigrateResourceTables()
		model.MigrateHistoryTables()
	}
}

func (c *Compass) Run() error {
	var err error

	c.RunXdsManagerServer()
	c.RunDashboardServer()
	c.RunPrometheusExporter()

	c.stopper.RunUntilStop(c.cancel)

	return err
}

func (c *Compass) RunXdsManagerServer() {

	listener, err := net.Listen(`tcp`, c.options.XdsManagerListen)
	if err != nil {
		logs.Logger.WithError(err).Fatal("xds management listen error")
	}

	grpcServer := grpc.NewServer()
	xdsSrv := xds.NewServer(c.done, c.cache, &callback.XdsStreamCallbacks{})
	xdiscovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdsSrv)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, xdsSrv)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, xdsSrv)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, xdsSrv)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, xdsSrv)

	logs.Logger.WithField("addr", c.options.XdsManagerListen).Info("xds management server listening")

	go func() {
		err = grpcServer.Serve(listener)
		if err != nil {
			logs.Logger.WithError(err).Fatal("xds management gRPC server error")
		}
	}()

	c.stopper.RegistryExitHook(`xds manager`, func() error {
		// FIXME: fuck !!!!!!
		//grpcServer.GracefulStop()
		grpcServer.Stop()
		return nil
	})
}

func (c *Compass) RunDashboardServer() {

	listener, err := dashboard.NewListenWithTryTime(c.options.DashboardListen, time.Second)
	if err != nil {
		logs.Logger.WithError(err).Fatal("create listener error")
	}

	logs.Logger.WithField("addr", c.options.DashboardListen).Info("http dashboard server listening")

	go func() {

		service.LoadDashboardTemplates()

		err = fasthttp.Serve(listener, dashboard.HandleDashBoardRequest)
		if err != nil {
			logs.Logger.WithError(err).Fatal("http dashboard serve error")
			return
		}
	}()

	c.stopper.RegistryExitHook(`dashboard`, func() error {
		return listener.Close()
	})
}

func (c *Compass) RunPrometheusExporter() {
	config := c.options.Metrics

	mux := http.NewServeMux()
	mux.Handle(config.URI, promhttp.Handler())

	srv := http.Server{
		Addr:         config.Listen,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		Handler:      mux,
	}

	go func() {
		logs.Logger.WithField("addr", config.Listen).Info("prometheus exporter server listening")
		_ = srv.ListenAndServe()
	}()

	c.stopper.RegistryExitHook(`metrics`, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	})
}
