package callback

import (
	"context"
	"git.sogou-inc.com/iweb/jstio/model"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/peer"
	"strings"

	. "git.sogou-inc.com/iweb/jstio/internel/logs"
)

type XdsStreamCallbacks struct {
}

func humanTyped(reqType string) string {
	if reqType == "" {
		return "None"
	}

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
		"stream_id":     streamID,
		"type":          humanTyped(req.TypeUrl),
		"version":       req.VersionInfo,
		"resp_nonce":    req.ResponseNonce,
		"error":         req.ErrorDetail,
		"resource_name": req.ResourceNames,
	}).Infoln("on stream request:", req.Node.Id)

	return nil
}

func (x *XdsStreamCallbacks) OnStreamResponse(streamID int64, req *v2.DiscoveryRequest, resp *v2.DiscoveryResponse) {
	Logger.WithFields(logrus.Fields{
		"stream_id":    streamID,
		"type":         humanTyped(req.TypeUrl),
		"resp_type":    humanTyped(resp.TypeUrl),
		"req_version":  req.VersionInfo,
		"resp_version": resp.VersionInfo,
		"req_nonce":    req.ResponseNonce,
		"resp_nonce":   resp.Nonce,
		"req_error":    req.ErrorDetail,
	}).Infoln("on stream response")
}

func (x *XdsStreamCallbacks) OnFetchRequest(ctx context.Context, req *v2.DiscoveryRequest) error {
	return nil
}

func (x *XdsStreamCallbacks) OnFetchResponse(req *v2.DiscoveryRequest, resp *v2.DiscoveryResponse) {
}
