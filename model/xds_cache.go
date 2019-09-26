package model

import (
	"errors"
	"fmt"
	"github.com/mitchellh/hashstructure"
	"jstio/internel"
	. "jstio/internel/logs"
	"strings"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
)

type XdsCache struct {
	cache.SnapshotCache

	discovery *Discovery
	builder   *ResourceBuilder

	pubSub *PubSub

	stop <-chan struct{}
}

func MustNewXdsCache(stop <-chan struct{}) *XdsCache {

	tagLog := FuncTaggedLoggerFactory()

	meta := internel.GetAfxMeta()

	discovery, err := NewDiscoveryWithPrefix(meta.ETCDEndpoints, WatchKeyPrefix())
	if err != nil {
		tagLog("discovery").Panic(err)
	}

	xdsCache := &XdsCache{
		discovery: discovery,
		builder:   MustNewResourceBuilder(meta.ClusterName, discovery),
		stop:      stop,
	}

	nsqds, looupds := meta.NSQDEndpoints()
	xdsCache.pubSub, err = NewPubSub(nsqds, looupds, xdsCache)
	if err != nil {
		tagLog("pubsub").Errorln(err)
	}

	pushMode := internel.GetAfxMeta().PushMode
	xdsCache.SnapshotCache = cache.NewSnapshotCache(pushMode == AdsMode, xdsCache, nil)

	if err = xdsCache.preload(); err != nil {
		tagLog("preload").Panic(err)
	}

	if err = xdsCache.discovery.WatchEndpoints(); err != nil {
		tagLog("watch").Panic(err)
	}

	xdsCache.HandleEvent()

	return xdsCache
}

func (xc *XdsCache) ID(node *core.Node) string {

	nodeMeta := node.Metadata.GetFields()

	appName := node.Cluster
	odinCluster := nodeMeta["odin_cluster"].GetStringValue()

	// TODO: maybe used feature ...
	/*********************************************************
		namespace := nodeMeta["namespace"].GetStringValue()
		environment := nodeMeta["env"].GetStringValue()
		return strings.Join([]string{appName, odinCluster, namespace, environment}, ".")
	**********************************************************/

	return strings.Join([]string{appName, odinCluster, `odin`, `sogou`}, ".")
}

func (xc *XdsCache) buildXdsResource(app *Application) (XdsResource, error) {
	var err error

	tagLog := FuncTaggedLoggerFactory()

	xdsRes := XdsResource{}
	for _, res := range app.Resources {
		xRes, err := xc.builder.BuildCustomResource(app, res.ResType, []byte(res.Config))
		if err != nil {
			tagLog(res.ResType).Errorln("app:", app.Hash(), ",error:", err)
			continue
		}

		switch res.ResType {
		case ResourceTypeRoute:
			xdsRes.Routers = xRes
		case ResourceTypeCluster:
			xdsRes.Clusters = xRes
		case ResourceTypeEndpoint:
			// FIXME: delete this use `buildFastAppEndpoints`
			xdsRes.Endpoints = xRes
		case ResourceTypeListener:
			xdsRes.Listener = xRes
		}
	}

	if xdsRes.Clusters == nil {
		xdsRes.Clusters, _ = xc.builder.BuildFastAppCluster(app)
	}

	if xdsRes.Endpoints == nil {
		ends, err := xc.builder.BuildFastAppEndpoints(app)
		if err != nil {
			tagLog("build endpoints").Errorln("app:", app.Hash(), ",error:", err)
		}
		xdsRes.Endpoints = ends
	}

	if xdsRes.Routers == nil {
		xdsRes.Routers, _ = xc.builder.BuildFastAppRoutes(app)
	}

	if xdsRes.Listener == nil {
		xdsRes.Listener, _ = xc.builder.BuildHTTPListener(app)
	}

	return xdsRes, err
}

func (xc *XdsCache) SetApplication(app *Application) (string, error) {
	var version string

	if app == nil {
		return version, errors.New("invalid param")
	}

	xdsRes, err := xc.buildXdsResource(app)
	if err != nil {
		return version, err
	}

	version = xc.ResVersion(xdsRes)
	snapshot := cache.NewSnapshot(version, xdsRes.Endpoints, xdsRes.Clusters, xdsRes.Routers, xdsRes.Listener)
	if err = snapshot.Consistent(); err != nil {
		return version, err
	}
	err = xc.SetSnapshot(app.Hash(), snapshot)
	return version, err
}

func (xc *XdsCache) MeshEndpointRefresh(app Selector) error {
	var err error

	tagLog := FuncTaggedLoggerFactory()

	appHash := app.Hash()

	defer func() {
		if err != nil {
			tagLog(appHash).Errorln(err)
		} else {
			tagLog(appHash).Println("success")
		}
	}()

	//activeApps, err := CompleteActiveApps()
	activeApps := GetApplicationCache().GetActiveApplications()

	completeApp, ok := activeApps[appHash]
	if ok {
		if err = xc.appEndpointRefresh(completeApp); err != nil {
			return err
		}

		for _, a := range completeApp.Downstream {
			err = xc.appEndpointRefresh(activeApps[a.Hash()])
		}

	} else {
		err = fmt.Errorf("no such registry app: %s", appHash)
	}

	return err
}

func (xc *XdsCache) appEndpointRefresh(app *Application) error {
	if app == nil {
		return errors.New("invalid param")
	}

	tagLog := FuncTaggedLoggerFactory()
	appHash := app.Hash()

	snapshot, err := xc.GetSnapshot(appHash)
	if err != nil {
		return err
	}

	//res, _ := xc.builder.BuildFastAppEndpoints(app)
	res, _ := xc.builder.BuildCustomAppEndpoints(app)

	oldEndpointVersion, newEndpointVersion := snapshot.GetVersion(cache.EndpointType), xc.ResVersion(res)

	if oldEndpointVersion == newEndpointVersion {
		tagLog("version compare").Println("app:", appHash, " same version")
		return nil
	}

	tagLog("version compare").Warning("app:", appHash, oldEndpointVersion, "!=", newEndpointVersion)

	snapshot.Endpoints = cache.NewResources(newEndpointVersion, res)

	if err = snapshot.Consistent(); err != nil {
		return err
	}

	return xc.SetSnapshot(appHash, snapshot)
}

func (xc *XdsCache) preload() error {
	var err error

	tagLog := FuncTaggedLoggerFactory()

	//activeApps, err := CompleteActiveApps()
	//if err != nil {
	//	return err
	//}
	activeApps := GetApplicationCache().GetActiveApplications()

	for appHash, app := range activeApps {
		xdsRes, err := xc.buildXdsResource(app)
		if err != nil {
			tagLog("build resource").Errorln("app:", appHash, ",error:", err)
		}

		// FIXME: use endpoint as init version.
		version := xc.ResVersion(xdsRes.Endpoints)
		snapshot := cache.NewSnapshot(version, xdsRes.Endpoints, xdsRes.Clusters, xdsRes.Routers, xdsRes.Listener)

		// FIXME: maybe request flood ???
		if err = snapshot.Consistent(); err != nil {
			tagLog("snapshot consistent").Errorln("app:", appHash, ",error:", err)
			continue
		}

		_ = xc.SetSnapshot(appHash, snapshot)

	}

	return err
}

func (xc *XdsCache) ResVersion(res ...interface{}) string {
	if len(res) == 0 {
		return xc.dummyVersion(errors.New("invalid param"))
	}

	vi, err := hashstructure.Hash(res, nil)
	if err != nil {
		return xc.dummyVersion(err)
	}

	return fmt.Sprintf("%020d", vi)
}

func (xc *XdsCache) dummyVersion(err error) string {
	Logger.Errorln("oops: should never be here:", err)
	return fmt.Sprintf("%020s", time.Now().Format("2006-01-02 15:04"))
}

func (xc *XdsCache) HandleEvent() {

	tagLog := FuncTaggedLoggerFactory()

	eventNotifier := GetEventNotifier()

	whoImI := internel.GetAfxMeta().Node

	go func() {
		for {
			select {
			case d := <-eventNotifier.Bus():
				if d.E == EventResEndpointReFetch {
					app, ok := d.Data.(*Application)
					if !ok {
						tagLog("assert").Errorln("assert failed")
						continue
					}

					_ = xc.MeshEndpointRefresh(app)

				} else if d.E > EventNone && d.E <= EventResEndpointDelete {
					app, ok := d.Data.(*Application)
					if !ok {
						tagLog("assert").Errorln("event", d.E)
						continue
					}

					version, err := xc.SetApplication(app)
					if err != nil {
						tagLog("set snapshot").Errorln("app:", app.Hash(), ",error:", err)
					} else {

						msg := XdsClusterMsg{
							MsgType: int(d.E),
							Sender:  whoImI,
							Version: version,
							Payload: app.ID,
						}

						if err = xc.pubSub.Publish(&msg); err != nil {
							tagLog("publish").Errorln("app:", app.Hash(), ",error:", err)
						} else {
							tagLog("publish").Println("app:", app.Hash(), " success")
						}
					}
				}
			case <-xc.stop:
				xc.pubSub.DeleteChannel()
				tagLog("stop").Warning("will stopped")
				return
			}
		}
	}()
}

func (xc *XdsCache) HandleClusterMsg(msg *XdsClusterMsg) {
	tagLog := FuncTaggedLoggerFactory()

	if msg.Sender == internel.GetAfxMeta().Node {
		tagLog("self").Println("pass")
		return
	}

	GetApplicationCache().ReBuild()

	appId, ok := msg.Payload.(float64)
	if !ok || appId == 0 {
		tagLog("assert payload").Errorln("invalid payload")
		return
	}

	app, err := GetApplicationById(uint(appId))
	if err != nil {
		tagLog("get app").Errorln(err)
		return
	}

	xdsRes, err := xc.buildXdsResource(&app)
	if err != nil {
		tagLog("build resource").Errorln(err)
		return
	}

	version := xc.ResVersion(xdsRes)
	if version != msg.Version { // FIXME: miss match
		tagLog("version mismatch").Errorln("local version:", version, ",msg version:", msg.Version)
		return
	}

	snapshot := cache.NewSnapshot(version, xdsRes.Endpoints, xdsRes.Clusters, xdsRes.Routers, xdsRes.Listener)
	if err = snapshot.Consistent(); err != nil {
		tagLog("consistent").Errorln(err)
		return
	}

	err = xc.SetSnapshot(app.Hash(), snapshot)
	if err != nil {
		tagLog("set snapshot").Errorln(err)
	}

	tagLog("done").Println(app.Hash(), "update success")
}
