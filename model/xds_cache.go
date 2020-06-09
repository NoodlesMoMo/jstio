package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"git.sogou-inc.com/iweb/jstio/internel"
	"git.sogou-inc.com/iweb/jstio/internel/logs"
	"git.sogou-inc.com/iweb/jstio/internel/util"
	"github.com/golang/protobuf/proto"
	"github.com/mitchellh/hashstructure"
	"os"
	"strings"
	"time"
	"unsafe"

	xcore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/mitchellh/mapstructure"
)

type XdsAppEventPayLod struct {
	AppID uint `json:"app_id"`
	Event int  `json:"event"`
}

func (xap *XdsAppEventPayLod) Convert(d map[string]interface{}) error {
	if appID, err := xap.fetchKeyInt64("app_id", d); err != nil {
		return err
	} else {
		xap.AppID = uint(appID)
	}

	if event, err := xap.fetchKeyInt64("event", d); err != nil {
		return err
	} else {
		xap.Event = int(event)
	}

	if xap.AppID == 0 || xap.Event == EventNone {
		return errors.New("zero data")
	}

	return nil
}

func (xap *XdsAppEventPayLod) fetchKeyInt64(key string, d map[string]interface{}) (int64, error) {
	v, ok := d[key]
	if !ok {
		return 0, errors.New("key error")
	}

	vv, ok := v.(json.Number)
	if !ok {
		return 0, errors.New("json number type error")
	}

	return vv.Int64()
}

type XdsCache struct {
	cache.SnapshotCache
	node string

	discovery *Discovery
	builder   *ResourceBuilder
	pubSub    *PubSub

	done context.Context
}

func MustNewXdsCache(done context.Context) *XdsCache {

	tagLog := logs.FuncTaggedLoggerFactory()

	options := internel.GetAfxOption()

	discovery, err := NewDiscovery(options.GetETCDEndpoints(), WatchKeyPrefix())
	if err != nil {
		tagLog("discovery").Panic(err)
	}

	xdsCache := &XdsCache{
		discovery: discovery,
		builder:   MustNewResourceBuilder(options.XdsManagerName, discovery),
		done:      done,
		node: func() string {
			hostName, _ := os.Hostname()
			return hostName + ":" + util.GetLocalIPV4Addr()
		}(),
	}

	xdsCache.pubSub, err = NewPubSub(options.NSQLookupdAddress, options.NSQTopic, xdsCache)
	if err != nil {
		tagLog("pubsub").Errorln(err)
	}

	xdsCache.SnapshotCache = cache.NewSnapshotCache(true, xdsCache, logs.Logger)

	if err = xdsCache.preload(); err != nil {
		tagLog("preload").Panic(err)
	}

	go xdsCache.discovery.WatchEndpoints()

	xdsCache.HandleEvent()

	return xdsCache
}

func (xc *XdsCache) ID(node *xcore.Node) string {

	nodeMeta := node.Metadata.GetFields()

	appName := node.Cluster
	odinCluster := nodeMeta["odin_cluster"].GetStringValue()

	// TODO: maybe used feature ...
	/*********************************************************
		namespace := nodeMeta["namespace"].GetStringValue()
		environment := nodeMeta["env"].GetStringValue()
		return strings.Join([]string{appName, odinCluster, namespace, environment}, ".")
	**********************************************************/

	return strings.Join([]string{appName, odinCluster, SOUGOSLD}, ".")
}

func (xc *XdsCache) GetCloneSnapshot(node string) (cache.Snapshot, error) {
	src, err := xc.GetSnapshot(node)
	if err != nil {
		return src, err
	}

	itemCopier := func(items map[string]cache.Resource) map[string]cache.Resource {
		cpy := make(map[string]cache.Resource, len(items))
		for k, v := range items {
			resCpy := proto.Clone(v)
			cpy[k] = resCpy
		}
		return cpy
	}

	snapshotCpy := cache.Snapshot{}
	snapshotCpy.Endpoints = cache.Resources{
		Version: src.Endpoints.Version,
		Items:   itemCopier(src.Endpoints.Items),
	}
	snapshotCpy.Clusters = cache.Resources{
		Version: src.Clusters.Version,
		Items:   itemCopier(src.Clusters.Items),
	}
	snapshotCpy.Routes = cache.Resources{
		Version: src.Routes.Version,
		Items:   itemCopier(src.Routes.Items),
	}
	snapshotCpy.Listeners = cache.Resources{
		Version: src.Listeners.Version,
		Items:   itemCopier(src.Listeners.Items),
	}

	return snapshotCpy, nil
}

func (xc *XdsCache) buildXdsResource(app *Application) (XdsResource, error) {
	var err error

	tagLog := logs.FuncTaggedLoggerFactory()

	xdsRes := XdsResource{}
	for _, res := range app.Resources {
		xRes, err := xc.builder.BuildCustomResource(app, res.ResType, *(*[]byte)(unsafe.Pointer(&res.Config)))
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
			xdsRes.Endpoints = xRes
		case ResourceTypeListener:
			xdsRes.Listener = xRes
		}
	}

	return xdsRes, err
}

func (xc *XdsCache) UpdateAppXdsSnapshot(app *Application, e Event) (string, error) {
	var (
		version = "empty-version"
	)

	if app == nil || e == EventNone {
		return version, errors.New("invalid param")
	}

	if e == EventApplicationCreate {
		return xc.NewApplicationSnapshot(app)
	}

	hash := app.Hash()
	snapshot, err := xc.GetSnapshot(hash)
	if err != nil {
		return version, fmt.Errorf("app:%s get snapshot error", hash)
	}

	xdsRes, err := xc.buildXdsResource(app)
	if err != nil {
		return version, err
	}

	switch e {
	case
		/* endpoint changed */
		EventResEndpointCreate,
		EventResEndpointUpdate,
		EventResEndpointDelete,
		/* cluster changed */
		EventResClusterCreate,
		EventResClusterUpdate,
		EventResClusterDelete:
		version = xc.ResVersion(xdsRes.Endpoints, xdsRes.Clusters)
		snapshot.Endpoints = cache.NewResources(version, xdsRes.Endpoints)
		snapshot.Clusters = cache.NewResources(version, xdsRes.Clusters)
	case
		EventResRouteCreate,
		EventResRouteUpdate,
		EventResRouteDelete:
		fallthrough
	case
		//EventApplicationCreate,
		EventApplicationUpdate,
		EventApplicationDelete:
		fallthrough
	case
		EventResListenerCreate,
		EventResListenerUpdate,
		EventResListenerDelete:
		fallthrough
	default:
		version = xc.ResVersion(xdsRes)
		snapshot.Endpoints = cache.NewResources(version, xdsRes.Endpoints)
		snapshot.Clusters = cache.NewResources(version, xdsRes.Clusters)
		snapshot.Routes = cache.NewResources(version, xdsRes.Routers)
		snapshot.Listeners = cache.NewResources(version, xdsRes.Listener)
	}

	err = xc.SetSnapshot(hash, snapshot)

	return version, err
}

func (xc *XdsCache) MeshEndpointRefresh(app Selector) error {
	var err error

	tagLog := logs.FuncTaggedLoggerFactory()

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
		return errors.New("invalid app param")
	}
	tagLog := logs.TaggedLoggerFactory("app endpoint refresh")
	appHash := app.Hash()

	snapshot, err := xc.GetSnapshot(appHash)
	if err != nil {
		return err
	}

	res, _ := xc.builder.BuildCustomAppEndpoints(app)

	oldEndpointVersion, newEndpointVersion := snapshot.GetVersion(cache.EndpointType), xc.ResVersion(res)
	if oldEndpointVersion == newEndpointVersion {
		tagLog("version compare").Println("app:", appHash, " same version")
		return nil
	}

	tagLog("version compare").Warningln("app:", appHash, oldEndpointVersion, " != ", newEndpointVersion)

	snapshot.Endpoints = cache.NewResources(newEndpointVersion, res)

	if err = snapshot.Consistent(); err != nil {
		return err
	}

	return xc.SetSnapshot(appHash, snapshot)
}

func (xc *XdsCache) preload() error {
	var err error

	tagLog := logs.FuncTaggedLoggerFactory()

	activeApps := GetApplicationCache().GetActiveApplications()

	for appHash, app := range activeApps {
		if _, err = xc.NewApplicationSnapshot(app); err != nil {
			tagLog("build snapshot").Errorln("app:", appHash, "error:", err)
		}
	}

	return err
}

func (xc *XdsCache) NewApplicationSnapshot(app *Application) (version string, err error) {
	tagLog := logs.TaggedLoggerFactory("new application snapshot")
	xdsRes, err := xc.buildXdsResource(app)
	appHash := app.Hash()

	if err != nil {
		tagLog("build resource").Errorln("app:", appHash, ",error:", err)
		return
	}

	// FIXME: use listener as init version ?
	version = xc.ResVersion(xdsRes.Listener)
	snapshot := cache.NewSnapshot(version, xdsRes.Endpoints, xdsRes.Clusters, xdsRes.Routers, xdsRes.Listener, nil)

	// FIXME: maybe request flood ???
	if err = snapshot.Consistent(); err != nil {
		tagLog("snapshot consistent").Errorln("app:", appHash, ",error:", err)
		return
	}

	err = xc.SetSnapshot(appHash, snapshot)
	return
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
	logs.Logger.Errorln("oops: should never be here:", err)
	return fmt.Sprintf("%020s", time.Now().Format("2006-01-02 15:04"))
}

func (xc *XdsCache) HandleEvent() {

	tagLog := logs.FuncTaggedLoggerFactory()

	eventNotifier := GetEventNotifier()

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

				} else if d.E > EventNone && d.E < EventResEndpointReFetch {
					app, ok := d.Data.(*Application)
					if !ok {
						tagLog("assert").Errorln("event", d.E)
						continue
					}

					version, err := xc.UpdateAppXdsSnapshot(app, d.E)
					if err != nil {
						tagLog("set snapshot").Errorln("app:", app.Hash(), ",error:", err)
					} else {

						msg := XdsClusterMsg{
							MsgType: int(d.E),
							Sender:  xc.node,
							Version: version,
							Payload: XdsAppEventPayLod{
								AppID: app.ID,
								Event: d.E,
							},
						}

						if err = xc.pubSub.Publish(&msg); err != nil {
							tagLog("publish").Errorln("app:", app.Hash(), ",error:", err)
						} else {
							tagLog("publish").Println("app:", app.Hash(), " success")
						}
					}
				}
			case <-xc.done.Done():
				xc.pubSub.DeleteChannel()
				tagLog("stop").Warning("will stopped")
				return
			}
		}
	}()
}

func (xc *XdsCache) HandleClusterMsg(msg *XdsClusterMsg) {
	tagLog := logs.TaggedLoggerFactory("[NSQ-MESSAGE]")

	if msg.Sender == xc.node {
		tagLog("self").Println("pass")
		return
	}

	if msg.Sender == PodCleanerSender {
		md, ok := msg.Payload.(map[string]interface{})
		if !ok {
			tagLog("pod_cleaner").Errorln("invalid payload")
		} else {
			tagLog("pod_cleaner").Println(md["hash"])
			pw := PodWait{}
			if err := mapstructure.Decode(md, &pw); err == nil {
				ct := md["created_at"].(string)
				if t, e := time.Parse(time.RFC3339, ct); e == nil {
					pw.CreatedAt = t
					appWaitCache.Put(&pw)

					if GetEventNotifier().Push(EventResEndpointReFetch, pw.ToApplication()) == nil {
						tagLog("endpoint deleting").Println("success:", pw.Hash, "ip:", pw.Addr)
					}
				}
			}
		}
		return
	}

	GetApplicationCache().ReBuild()

	tag := `nsq-consume-app-event`
	md, ok := msg.Payload.(map[string]interface{})
	if !ok {
		tagLog(tag).Errorln("invalid payload")
		return
	}

	payload := XdsAppEventPayLod{}
	if err := payload.Convert(md); err != nil {
		tagLog(tag).WithError(err).Errorln("convert error")
		return
	}

	app, err := GetApplicationByID(payload.AppID)
	if err != nil {
		tagLog("get app").Errorln(err)
		return
	}

	appHash := app.Hash()
	version, err := xc.UpdateAppXdsSnapshot(&app, payload.Event)
	if err != nil {
		tagLog("update xds snapshot").WithError(err).Errorln("update snapshot failed! app:", appHash)
		return
	}
	if version != msg.Version {
		tagLog("version mismatch").Errorln("local version:", version, "message version:", msg.Version, "app:", appHash)
	}

	tagLog("done").Println(app.Hash(), "update success")
}
