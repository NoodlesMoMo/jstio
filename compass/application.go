package compass

import (
	"jstio/compass/callback"
	"jstio/dashboard"
	"jstio/internel"
	. "jstio/internel/logs"
	"jstio/model"
	"net"
	"time"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	xdiscovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
)

type noCopy struct{}

func (*noCopy) Lock() {}

type Compass struct {
	noCopy

	*core.Node

	done    chan struct{}
	meta    *internel.AfxMetaData
	cache   *model.XdsCache
	stopper *internel.GracefulStopper
}

func NewCompass() *Compass {

	compass := &Compass{
		done:    make(chan struct{}),
		stopper: internel.NewGracefulStopper(),
	}

	if err := compass.initialization(); err != nil {
		Logger.Panic(err)
	}

	return compass
}

func (c *Compass) initialization() error {
	var err error

	// load metadata from config file
	c.meta = internel.GetAfxMeta()

	c.autoMigrateTables(c.meta.DebugMode)

	c.cache = model.MustNewXdsCache(c.done)

	c.stopper.RegistryExitHook(`internal done control`, func() error {
		close(c.done)
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

	c.stopper.RunUntilStop(Logger)

	return err
}

func (c *Compass) RunXdsManagerServer() {

	listener, err := net.Listen(`tcp`, c.meta.XdsManagerListen)
	if err != nil {
		Logger.WithError(err).Fatal("xds management listen error")
	}

	grpcServer := grpc.NewServer()
	xdsSrv := xds.NewServer(c.cache, &callback.XdsStreamCallbacks{})
	//algSrv := &AccessLogService{}
	//xaccesslog.RegisterAccessLogServiceServer(grpcServer, algSrv)
	xdiscovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdsSrv)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, xdsSrv)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, xdsSrv)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, xdsSrv)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, xdsSrv)

	Logger.WithField("addr", c.meta.XdsManagerListen).Info("xds management server listening")

	go func() {
		err = grpcServer.Serve(listener)
		if err != nil {
			Logger.WithError(err).Fatal("xds management gRPC server error")
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

	listener, err := dashboard.NewListenWithTryTime(c.meta.DashboardListen, time.Second)
	if err != nil {
		Logger.WithError(err).Fatal("create listener error")
	}

	Logger.WithField("addr", c.meta.DashboardListen).Info("http dashboard server listening")

	go func() {
		err = fasthttp.Serve(listener, dashboard.HandleDashBoardRequest)
		if err != nil {
			Logger.WithError(err).Fatal("http dashboard serve error")
			return
		}
	}()

	c.stopper.RegistryExitHook(`dashboard`, func() error {
		return listener.Close()
	})
}
