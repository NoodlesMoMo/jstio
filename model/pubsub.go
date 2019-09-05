package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nsqio/go-nsq"
	"github.com/valyala/fasthttp"
	"go.uber.org/atomic"
	"jstio/internel"
	. "jstio/internel/logs"
	"jstio/internel/util"
	"net/url"
	"os"
	"time"
)

const (
	jstioTopic = `jstio_cluster_sys`
)

type XdsClusterMsg struct {
	MsgType int         `json:"msg_type"`
	Sender  string      `json:"sender"`
	Version string      `json:"version"`
	Payload interface{} `json:"payload"`
}

func (m *XdsClusterMsg) Marshal() ([]byte, error) {
	if m.MsgType == 0 || m.Sender == "" || m.Version == "" || m.Payload == nil {
		return nil, errors.New("invalid message")
	}

	return json.Marshal(m)
}

type XdsClusterHandler interface {
	HandleClusterMsg(*XdsClusterMsg)
}

type PubSub struct {
	brokerAddr     []string
	brokerHTTPAddr []string
	topic          string
	channel        string
	producers      []*nsq.Producer
	consumer       *nsq.Consumer
	pubMsgCnt      *atomic.Int64
	handler        XdsClusterHandler
}

func NewPubSub(nsqds, nsqdHTTPAddr []string, handler_ XdsClusterHandler) (*PubSub, error) {
	var err error

	inst := &PubSub{
		brokerAddr:     nsqds,
		brokerHTTPAddr: nsqdHTTPAddr,
		pubMsgCnt:      atomic.NewInt64(0),
		topic:          jstioTopic,
		handler:        handler_,
	}

	cfg := nsq.NewConfig()
	for _, addr := range inst.brokerAddr {
		if p, e := nsq.NewProducer(addr, cfg); e != nil {
			err = e
			Logger.WithField(`PubSub::NewProducer`, addr).Errorln(e)
		} else {
			inst.producers = append(inst.producers, p)
		}
	}

	consumeConfig := nsq.NewConfig()
	consumeConfig.MaxInFlight = 32

	hostName, _ := os.Hostname()
	inst.channel = hostName + "_" + util.GetLocalIPV4Addr()
	inst.consumer, err = nsq.NewConsumer(jstioTopic, inst.channel, consumeConfig)
	inst.consumer.AddHandler(inst)
	inst.consumer.SetLogger(NSQLogger{}, nsq.LogLevelInfo)

	//err = inst.consumer.ConnectToNSQDs(inst.brokerAddr)
	err = inst.consumer.ConnectToNSQLookupd(internel.GetAfxMeta().NSQCluster.LookupAddress)

	return inst, err
}

func (ps *PubSub) DeleteChannel() {
	tagLog := FuncTaggedLoggerFactory()

	params := url.Values{}
	params.Add("topic", ps.topic)
	params.Add("channel", ps.channel)

	for _, addr := range ps.brokerHTTPAddr {
		uri := fmt.Sprintf("http://%s/channel/delete?%s", addr, params.Encode())
		ps.Post(uri, time.Second)
	}

	ps.consumer.Stop()

	tagLog("release channel").Println(ps.topic, ps.channel, "success")
}

func (ps *PubSub) Post(url string, timeout time.Duration) {
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(url)
	req.Header.SetMethod(`POST`)

	_ = fasthttp.DoTimeout(req, resp, timeout)
}

func (ps *PubSub) Publish(msg *XdsClusterMsg) error {
	var err error

	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	pLen := int64(len(ps.producers))
	for i := 0; i < 3; i++ {
		cnt := ps.pubMsgCnt.Load()
		ps.pubMsgCnt.Inc()
		producer := ps.producers[cnt%pLen]
		if err = producer.Publish(ps.topic, data); err == nil {
			break
		}
	}

	return err
}

func (ps *PubSub) HandleMessage(msg *nsq.Message) error {
	tagLog := FuncTaggedLoggerFactory()

	tagLog("message").Println(string(msg.Body))

	if msg.Body == nil || len(msg.Body) == 0 {
		return nil
	}

	xdsMsg := XdsClusterMsg{}
	if err := json.Unmarshal(msg.Body, &xdsMsg); err != nil {
		tagLog("unmarshal").Errorln(err)
	}

	ps.handler.HandleClusterMsg(&xdsMsg)

	return nil
}
