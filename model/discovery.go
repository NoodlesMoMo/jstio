package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "jstio/internel/logs"

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
	prefix string
}

func NewDiscovery(endpoints []string) (*Discovery, error) {
	return NewDiscoveryWithPrefix(endpoints, defaultRegistryPrefix)
}

func NewDiscoveryWithPrefix(endpoints []string, prefix string) (*Discovery, error) {
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
		Logger.WithField(`discovery`, `fetch endpoints`).Errorln(err)
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
			Logger.WithField(`discovery`, `fetch endpoints ex`).Errorln(err)
			continue
		}

		for _, node := range resp.Node.Nodes {
			value := strings.TrimSpace(node.Value)
			if value != "" {
				rk := selector.Hash()
				result[rk] = append(result[rk], Endpoint{
					Address: strings.Split(value, ":")[0],
					Port:    80,
				})
			}
		}
	}

	return result, err
}

func (d *Discovery) WatchEndpoints() error {
	var err error

	watcher := client.NewKeysAPI(d.cli).Watcher(d.prefix, &client.WatcherOptions{
		Recursive: true,
	})

	notifier := GetEventNotifier()

	go func() {
		for {
			res, err := watcher.Next(context.Background())
			if err != nil {
				break
			}

			if res.PrevNode != nil && res.PrevNode.Value == res.Node.Value {
				continue
			}

			app := &Application{}
			err = app.SelectorScan(res.Node.Key)
			if err != nil {
				Logger.WithField(`discovery`, `watch`).Errorln(err)
				continue
			}

			// TODO: do best later ...
			switch res.Action {
			case `set`, `update`, `expire`, `delete`:
				_ = notifier.Push(EventResEndpointReFetch, app)
			}
		}
	}()

	return err
}
