package model

//
//import (
//	"errors"
//	"jstio/internel/logs"
//	"sync"
//)
//
//var (
//	ErrCacheNotExist = errors.New("not exist")
//)
//
//var (
//	_appCache     *ApplicationCache
//	_appCacheOnce = sync.Once{}
//)
//
//type Iterator struct {
//	C    <-chan *Application
//	stop chan struct{}
//}
//
//func (i *Iterator) Stop() {
//	defer func() {
//		recover()
//	}()
//
//	close(i.stop)
//
//	for range i.C {
//	}
//}
//
//func newIterator() (*Iterator, chan<- *Application, <-chan struct{}) {
//	itemChan := make(chan *Application)
//	stopChan := make(chan struct{})
//
//	return &Iterator{
//		C:    itemChan,
//		stop: stopChan,
//	}, itemChan, stopChan
//}
//
//type ApplicationCache struct {
//	lock  sync.RWMutex
//	cache map[string]*Application
//}
//
//func GetApplicationCache() *ApplicationCache {
//	if _appCache != nil {
//		return _appCache
//	}
//
//	_appCacheOnce.Do(func() {
//		_appCache = &ApplicationCache{
//			lock: sync.RWMutex{},
//		}
//		_ = _appCache.Build() // FIXME: handle error
//	})
//
//	return _appCache
//}
//
//func (c *ApplicationCache) Get(hash string) (*Application, error) {
//	c.lock.RLock()
//	defer c.lock.RUnlock()
//
//	app, ok := c.cache[hash]
//	if ok {
//		return app, nil
//	}
//
//	return nil, ErrCacheNotExist
//}
//
//func (c *ApplicationCache) Set(app *Application) {
//	c.lock.Lock()
//	defer c.lock.Unlock()
//
//	c.cache[app.Hash()] = app
//}
//
//func (c *ApplicationCache) SetNX(app *Application) {
//	c.lock.Lock()
//	defer c.lock.Unlock()
//
//	_, ok := c.cache[app.Hash()]
//	if !ok {
//		c.cache[app.Hash()] = app
//	}
//}
//
//func (c *ApplicationCache) Reset() {
//	c.lock.Lock()
//	defer c.lock.Unlock()
//
//	c.cache = make(map[string]*Application)
//}
//
//func (c *ApplicationCache) Build() error {
//
//	tagLog := logs.FuncTaggedLoggerFactory()
//
//	apps, err := AllApps(true)
//	if err != nil {
//		tagLog("all apps").WithError(err).Errorln("get all applications error")
//		return err
//	}
//
//	c.lock.Lock()
//	defer c.lock.Unlock()
//
//	c.cache = make(map[string]*Application)
//	xcache := make(map[uint]*Application)
//
//	for _, app := range apps {
//		_app := app
//		c.cache[app.Hash()] = &_app
//		xcache[app.ID] = &_app
//	}
//
//	for _, app := range c.cache {
//		_app := app
//		up, down, err := GetAppRefersById(_app.ID)
//		if err != nil {
//			// FIXME: add error log
//			continue
//		}
//
//		for _, appId := range up {
//			_app.Upstream = append(_app.Upstream, xcache[appId])
//		}
//
//		for _, appId := range down {
//			_app.Downstream = append(_app.Downstream, xcache[appId])
//		}
//	}
//
//	return nil
//}
//
//func (c *ApplicationCache) Iterator() *Iterator {
//	iterator, dch, stop := newIterator()
//
//	go func() {
//		c.lock.Lock()
//		defer c.lock.Unlock()
//
//	Ldone:
//		for _, elem := range c.cache {
//			select {
//			case <-stop:
//				break Ldone
//			case dch <- elem:
//			}
//		}
//		close(dch)
//	}()
//
//	return iterator
//}
//
//func (c *ApplicationCache) ConstIterator() *Iterator {
//	iterator, dch, stop := newIterator()
//
//	go func() {
//		c.lock.Lock()
//		defer c.lock.Unlock()
//
//	Ldone:
//		for _, elem := range c.cache {
//			o := elem.UnsafeCopy()
//			select {
//			case <-stop:
//				break Ldone
//			case dch <- o:
//			}
//		}
//		close(dch)
//	}()
//
//	return iterator
//}
