package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"git.sogou-inc.com/iweb/jstio/internel"
	"git.sogou-inc.com/iweb/jstio/internel/logs"
	"git.sogou-inc.com/iweb/jstio/internel/util"
	"github.com/nsqio/go-nsq"
	"github.com/valyala/fasthttp"
	"go.uber.org/atomic"
	"net/url"
	"os"
	"time"
)

const (
	PodCleanerSender = `pod_cleaner`
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

type NSQNode struct {
	BroadcastAddress string `json:"broadcast_address"`
	TCPPort          int    `json:"tcp_port"`
	HTTPPort         int    `json:"http_port"`
}

type PubSub struct {
	nsqdAddrs     []string
	nsqdHTTPAddrs []string
	topic         string
	channel       string
	producers     []*nsq.Producer
	consumer      *nsq.Consumer
	pubMsgCnt     *atomic.Int64
	handler       XdsClusterHandler
}

func NewPubSub(lookupAddress string, topic string, handler_ XdsClusterHandler) (*PubSub, error) {
	var err error

	inst := &PubSub{
		pubMsgCnt: atomic.NewInt64(0),
		topic:     topic,
		handler:   handler_,
	}

	if err = inst.fetchNodes(lookupAddress); err != nil {
		return inst, err
	}

	cfg := nsq.NewConfig()
	for _, addr := range inst.nsqdAddrs {
		if p, e := nsq.NewProducer(addr, cfg); e != nil {
			err = e
			logs.Logger.WithField(`PubSub::NewProducer`, addr).Errorln(e)
		} else {
			inst.producers = append(inst.producers, p)
		}
	}

	consumeConfig := nsq.NewConfig()
	consumeConfig.MaxInFlight = 64

	hostName, _ := os.Hostname()
	inst.channel = hostName + "_" + util.GetLocalIPV4Addr()
	inst.consumer, err = nsq.NewConsumer(topic, inst.channel, consumeConfig)
	inst.consumer.AddHandler(inst)
	inst.consumer.SetLogger(logs.Logger, nsq.LogLevelWarning)

	err = inst.consumer.ConnectToNSQLookupd(internel.GetAfxOption().NSQLookupdAddress)

	return inst, err
}

func (ps *PubSub) fetchNodes(lookupdAddress string) error {
	code, resp, err := fasthttp.GetTimeout(nil, lookupdAddress+"/nodes", 3*time.Second)
	if err != nil {
		return err
	}

	if code != fasthttp.StatusOK {
		return fmt.Errorf("code: %d", code)
	}

	producers := make(map[string][]NSQNode)
	if err = json.Unmarshal(resp, &producers); err != nil {
		return err
	}

	nodes := producers["producers"]
	if nodes == nil || len(nodes) == 0 {
		return errors.New("no usable nsqd")
	}

	for _, node := range nodes {
		ps.nsqdAddrs = append(ps.nsqdAddrs, fmt.Sprintf("%s:%d", node.BroadcastAddress, node.TCPPort))
		ps.nsqdHTTPAddrs = append(ps.nsqdHTTPAddrs, fmt.Sprintf("http://%s:%d", node.BroadcastAddress, node.HTTPPort))
	}

	return nil
}

func (ps *PubSub) DeleteChannel() {
	tagLog := logs.FuncTaggedLoggerFactory()

	params := url.Values{}
	params.Add("topic", ps.topic)
	params.Add("channel", ps.channel)

	for _, addr := range ps.nsqdHTTPAddrs {
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
	if pLen == 0 {
		return errors.New("no usable nsqd")
	}

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
	tagLog := logs.FuncTaggedLoggerFactory()

	tagLog("nsq-message").Println(string(msg.Body))

	if msg.Body == nil || len(msg.Body) == 0 {
		return nil
	}

	// FIXME: don't support float in payload?
	decoder := json.NewDecoder(bytes.NewReader(msg.Body))
	decoder.UseNumber()

	xdsMsg := XdsClusterMsg{}
	if err := decoder.Decode(&xdsMsg); err != nil {
		tagLog("unmarshal").Errorln(err)
	}

	ps.handler.HandleClusterMsg(&xdsMsg)

	return nil
}
