package cluster

import (
	"fmt"
	. "jstio/internel/logs"
	"jstio/internel/util"
	"net"
	"os"
	"sync"
	"time"
)

const (
	ConnUnusable = iota + 1
	ConnPending
	ConnDisconnect
	ConnConnWell
)

type Peer struct {
	Addr string `yaml:"addr"`

	conn  net.Conn
	ch    chan interface{}
	state int
	lock  *sync.RWMutex
}

type Cluster struct {
	Self  string  `yaml:"self"`
	Peers []*Peer `yaml:"peers"`

	bc Broadcaster
}

func BuildClusters(addrs []string) *Cluster {
	hostName, _ := os.Hostname()
	inst := &Cluster{
		Self: hostName + ":" + util.GetLocalIPV4Addr(),
		bc:   NewBroadcaster(4096, nil),
	}

	for _, addr := range addrs {
		peer := NewPeer(addr)
		if err := peer.Connect(); err != nil {
			Logger.WithField("build-cluster-connect", addr).Errorln(err)
		}
		inst.bc.Register(peer.Data())
		inst.Peers = append(inst.Peers, peer)
	}

	return inst
}

func (info *Cluster) Close() {
	if info.bc != nil {
		info.bc.Close()
	}
}

func NewPeer(addr string) *Peer {
	peer := &Peer{
		Addr:  addr,
		state: ConnUnusable,
		ch:    make(chan interface{}),
		lock:  &sync.RWMutex{},
	}

	return peer
}

func (p *Peer) Data() chan<- interface{} {
	return p.ch
}

func (p *Peer) Connect() error {
	return p.reconnect()
}

func (p *Peer) reconnect() error {
	var err error

	p.lock.Lock()
	defer p.lock.Unlock()

	if p.state == ConnConnWell {
		_ = p.conn.Close()
	}

	conn, err := net.DialTimeout("tcp", p.Addr, 3*time.Second)
	if err == nil {
		p.state = ConnConnWell
		p.conn = conn
	} else {
		p.state = ConnDisconnect
	}

	return err
}

func (p *Peer) serve() {
	for {
		data := <-p.ch
		fmt.Println(">>>>", data)

		b, ok := data.([]byte)
		if !ok {
			continue
		}

		p.lock.RLock()
		if p.state == ConnConnWell {
			p.conn.Write(b)
		}
		p.lock.RUnlock()
	}
}

func (p *Peer) autoReconnect() {
	var err error

	for {
		p.lock.RLock()
		stage := p.state
		p.lock.RUnlock()

		if stage == ConnConnWell {
			Logger.WithField("cluster-peer-reconnect", p.Addr).Println("connection ok!")
			time.Sleep(time.Second)
			continue
		}

		if err = p.reconnect(); err != nil {
			Logger.WithField("cluster-peer-reconnect", p.Addr).Errorln(err)
		}

		time.Sleep(time.Second)
	}
}
