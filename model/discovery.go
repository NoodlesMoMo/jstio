package model

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"git.sogou-inc.com/iweb/jstio/internel/logs"

	"go.etcd.io/etcd/client"
)

const (
	defaultRegistryPrefix = `/conf/`
	defaultRegistrySuffix = `/upstream`
)

var (
	defaultFetchTimeout = 3 * time.Second
)

type Discovery struct {
	cli    client.Client
	prefix []string
}

func NewDiscovery(endpoints []string, prefix []string) (*Discovery, error) {
	var err error

	inst := &Discovery{
		prefix: prefix,
	}

	inst.cli, err = client.New(client.Config{
		Endpoints:               endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: defaultFetchTimeout,
	})

	return inst, err
}

func (d *Discovery) FetchEndpoints(selector Selector) ([]Endpoint, error) {
	var err error

	endpoints := make([]Endpoint, 0)

	ctx, cancel := context.WithTimeout(context.Background(), defaultFetchTimeout)
	defer cancel()

	resp, err := client.NewKeysAPI(d.cli).Get(ctx, selector.SelectorFormat(),
		&client.GetOptions{
			Sort:      true,
			Recursive: true,
		})

	if err != nil && !client.IsKeyNotFound(err) {
		logs.Logger.WithField(`discovery`, `fetch endpoints`).Errorln(err)
		return endpoints, err
	}

	if resp == nil {
		return endpoints, fmt.Errorf("not found: %s", err)
	}

	for _, node := range resp.Node.Nodes {
		value := strings.TrimSpace(node.Value)
		if value != "" {
			endpoint := Endpoint{}
			seps := strings.Split(value, ":")
			endpoint.Address = seps[0]
			endpoint.Port = 80

			// FIXME: appWait filter
			if appWaitCache.IsDeleting(selector.Hash(), endpoint.Address) {
				logs.Logger.WithField(`discovery`, `pod-wait`).Warningln("hash:", selector.Hash(), "addr:", endpoint.Address)
				continue
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints, err
}

func (d *Discovery) FetchEndpointsAgg(selectors ...Selector) (map[string][]Endpoint, error) {
	var (
		err    error
		result = make(map[string][]Endpoint)
	)

	ctx, cancel := context.WithTimeout(context.Background(), defaultFetchTimeout)
	defer cancel()

	options := client.GetOptions{
		Sort:      true,
		Recursive: true,
	}
	keyAPI := client.NewKeysAPI(d.cli)

	for _, selector := range selectors {
		resp, err := keyAPI.Get(ctx, selector.SelectorFormat(), &options)
		if err != nil || resp == nil {
			logs.Logger.WithField(`discovery`, `fetch endpoints ex`).Errorln(err)
			continue
		}

		for _, node := range resp.Node.Nodes {
			value := strings.TrimSpace(node.Value)
			if value != "" {
				rk := selector.Hash()
				addr := strings.Split(value, ":")[0]
				// FIXME: add appWait filter
				if appWaitCache.IsDeleting(rk, addr) {
					logs.Logger.WithField(`discovery`, `pod-wait`).Warningln("hash:", rk, " addr:", addr)
					continue
				}
				result[rk] = append(result[rk], Endpoint{
					Address: addr,
					Port:    80,
				})
			}
		}
	}

	return result, err
}

func (d *Discovery) WatchEndpoints() {
	tagLog := logs.FuncTaggedLoggerFactory()

	notifier := GetEventNotifier()

	wg := sync.WaitGroup{}

	wg.Add(len(d.prefix))

	for _, prefix := range d.prefix {
		watcher := client.NewKeysAPI(d.cli).Watcher(prefix, &client.WatcherOptions{
			Recursive: true,
		})

		go func(prefix string) {
			defer wg.Done()

			for {
				res, err := watcher.Next(context.Background())
				if err != nil {
					tagLog("watch goroutine").WithError(err).Fatalln("watch error, key:", prefix)
				}

				if res.PrevNode != nil && res.PrevNode.Value == res.Node.Value {
					continue
				}

				app := &Application{}
				err = app.SelectorScan(res.Node.Key)
				if err != nil {
					tagLog(`selector scan`).Errorln(err)
					continue
				}

				// TODO: do best later ...
				switch res.Action {
				case `set`, `update`, `expire`, `delete`:
					tagLog("watch action").Println(">>>>>action:", res.Action, res.Node.Key)
					_ = notifier.Push(EventResEndpointReFetch, app)
				}
			}
		}(prefix)
	}

	wg.Wait()
}
