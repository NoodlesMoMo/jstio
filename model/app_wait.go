package model

import (
	"encoding/json"
	"sync"
	"time"
)

// 实现这个的原因是因为,OP维护的ETCD存在5秒的定时刷新
// pod处在非`running`状态时，jstio本身无法立刻得到通知。
// 为了避免丢失请求，这里增加一个`app_wait`。类比TCP的
// TIME_WAIT。

var (
	appWaitCache = NewAppWaitCache()
)

type PodWait struct {
	AppName     string        `mapstructure:"app_name" json:"app_name"`
	OdinCluster string        `mapstructure:"odin_cluster" json:"odin_cluster"`
	Namespace   string        `mapstructure:"namespace" json:"namespace"`
	Addr        string        `mapstructure:"addr" json:"addr"`
	Hash        string        `mapstructure:"hash" json:"hash"`
	CreatedAt   time.Time     `json:"created_at"`
	Life        time.Duration `json:"life"`
}

func (p *PodWait) key() string {
	return p.Hash + ":" + p.Addr
}

func (p *PodWait) isExpire() bool {
	if p.Life == 0 {
		return false
	}

	return time.Now().Sub(p.CreatedAt) >= p.Life
}

func (p *PodWait) ToApplication() *Application {
	return &Application{
		AppName:     p.AppName,
		Namespace:   p.Namespace,
		OdinCluster: p.OdinCluster,
	}
}

type AppWaitCache struct {
	sync.RWMutex
	Items map[string]*PodWait
}

func NewAppWaitCache() *AppWaitCache {
	c := &AppWaitCache{
		Items: make(map[string]*PodWait),
	}

	go c.gc()

	return c
}

func (c *AppWaitCache) IsDeleting(hash, addr string) bool {
	key := hash + ":" + addr

	c.RLock()
	_, exist := c.Items[key]
	c.RUnlock()

	return exist
}

func (c *AppWaitCache) Put(pw *PodWait) {
	if pw.Life == 0 {
		return
	}

	now := time.Now()

	c.Lock()
	defer c.Unlock()

	old, ok := c.Items[pw.key()]
	if ok {
		old.CreatedAt = now
	} else {
		c.Items[pw.key()] = pw
	}
}

func (c *AppWaitCache) gc() {
	for {

		c.deleteExpiredKeys(c.scanExpiredKeys())

		time.Sleep(200 * time.Millisecond)
	}
}

func (c *AppWaitCache) deleteExpiredKeys(keys []string) {
	if len(keys) == 0 {
		return
	}

	c.Lock()
	defer c.Unlock()
	for _, key := range keys {
		delete(c.Items, key)
	}
}

func (c *AppWaitCache) scanExpiredKeys() (keys []string) {
	c.RLock()
	defer c.RUnlock()

	for k, v := range c.Items {
		if v.isExpire() {
			keys = append(keys, k)
		}
	}

	return keys
}

func PodWaitDump() []byte {
	appWaitCache.RLock()
	defer appWaitCache.RUnlock()

	data, _ := json.Marshal(appWaitCache)

	return data
}
