package callback

import (
	"context"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/peer"
	"jstio/model"
	"strings"

	. "jstio/internel/logs"
)

type XdsStreamCallbacks struct {
}

func humanTyped(reqType string) string {
	return strings.TrimPrefix(reqType, model.EnvoyTypePrefix)
}

func (x *XdsStreamCallbacks) OnStreamOpen(ctx context.Context, streamID int64, types string) error {

	p, ok := peer.FromContext(ctx)
	if !ok {
		Logger.WithFields(logrus.Fields{
			"stream_id": streamID,
			"type":      humanTyped(types),
		}).Infoln("on stream open: peer not ok")
		return nil
	}

	Logger.WithFields(logrus.Fields{
		"stream_id": streamID,
		"type":      humanTyped(types),
	}).Infoln("on stream open:", p.Addr.String())

	return nil
}

func (x *XdsStreamCallbacks) OnStreamClosed(streamID int64) {
	Logger.WithFields(logrus.Fields{
		"stream_id": streamID,
	}).Infoln("on stream closed")
}

func (x *XdsStreamCallbacks) OnStreamRequest(streamID int64, req *v2.DiscoveryRequest) error {

	Logger.WithFields(logrus.Fields{
		"stream_id": streamID,
		"type":      humanTyped(req.TypeUrl),
	}).Infoln("on stream request:", req.Node.Id)

	return nil
}

func (x *XdsStreamCallbacks) OnStreamResponse(streamID int64, req *v2.DiscoveryRequest, resp *v2.DiscoveryResponse) {
	Logger.WithFields(logrus.Fields{
		"stream_id": streamID,
		"type":      humanTyped(req.TypeUrl),
	}).Infoln("on stream response")
}

func (x *XdsStreamCallbacks) OnFetchRequest(ctx context.Context, req *v2.DiscoveryRequest) error {
	return nil
}

func (x *XdsStreamCallbacks) OnFetchResponse(req *v2.DiscoveryRequest, resp *v2.DiscoveryResponse) {
}
