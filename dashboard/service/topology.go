package service

import (
	"encoding/json"
	"fmt"
	"git.sogou-inc.com/iweb/jstio/internel"
	"git.sogou-inc.com/iweb/jstio/model"
	"github.com/valyala/fasthttp"
	"regexp"
	"time"
)

const (
	NodeIngress  = `ingress`
	NodeRelay    = `relay`
	NodeEndpoint = `endpoint`
)

var (
	zeroStats = NodeStats{
		Total:    0,
		HTTPCode: map[string]int64{"_total": 0},
		Timeout:  map[string]int64{"_total": 0},
	}

	blueReg = regexp.MustCompile(`-blue$|-blue\.|-green$|-green\.`)
)

type NodeStats struct {
	Total    int64            `json:"_total"`
	HTTPCode map[string]int64 `json:"non200"`
	Timeout  map[string]int64 `json:"timeout"`
}

type VisNode struct {
	ID     uint      `json:"id"`
	Domain string    `json:"domain"`
	Group  uint      `json:"group"`
	Label  string    `json:"label"`
	Type   string    `json:"type"`
	Stats  NodeStats `json:"stats"`
}

type VisEdge struct {
	From uint `json:"from"`
	To   uint `json:"to"`
}

type VisNetwork struct {
	Nodes []VisNode `json:"nodes"`
	Edges []VisEdge `json:"edges"`
}

func (vn *VisNetwork) TryMergeStats(app *model.Application, stats map[string]NodeStats) {
	for idx, v := range vn.Nodes {
		if v.Domain == app.Domain() {
			ns := vn.Nodes[idx].Stats
			if appStats, ok := stats[app.Domain()]; ok {
				appStats.Total += ns.Total
				for k, vi := range appStats.Timeout {
					if _, b := ns.Timeout[k]; b {
						ns.Timeout[k] += vi
					} else {
						ns.Timeout[k] = vi
					}
				}
				for k, vi := range appStats.HTTPCode {
					if _, b := ns.HTTPCode[k]; b {
						ns.HTTPCode[k] += vi
					} else {
						ns.HTTPCode[k] = vi
					}
				}
			}
			break
		}
	}
}

func nodeType(app *model.Application) string {
	upstreamLen, downstreamLen := app.UpstreamIDs.Len(), app.DownstreamIDs.Len()
	if upstreamLen > 0 && downstreamLen > 0 {
		return NodeRelay
	} else if upstreamLen > 0 {
		return NodeIngress
	} else if downstreamLen > 0 {
		return NodeEndpoint
	}

	return NodeEndpoint
}

// GenApplicationVisNetwork from, to datestyle string
func GenApplicationVisNetwork(from, to string) (VisNetwork, error) {
	network := VisNetwork{}

	apps, err := model.ApplicationTopology()
	if err != nil {
		return network, err
	}

	stats := make(map[string]NodeStats)
	option := internel.GetAfxOption()
	if option.StatsBackend != "" {
		backend := option.StatsBackend + fmt.Sprintf("?from=%s&to=%s", from, to)
		code, body, err := fasthttp.GetTimeout(nil, backend, time.Second*5)
		if err != nil {
			return network, err
		}
		if code != 200 {
			return network, err
		}

		err = json.Unmarshal(body, &stats)
		if err != nil {
			return network, err
		}
	}

	//replacer := func(in string)string {
	//	if strings.LastIndex(in, ".") > 0 {
	//		return "."
	//	}
	//	return ""
	//}

	for _, app := range apps {
		// FIXME: stupid
		needMerge := false
		origApp := app
		if needMerge = blueReg.MatchString(app.AppName); needMerge {
			app.AppName = blueReg.ReplaceAllString(app.AppName, "")
		}

		node := VisNode{
			ID:     app.ID,
			Group:  app.ID,
			Label:  app.AppName + "." + app.OdinCluster,
			Domain: app.Domain(),
			Type:   nodeType(&app),
			Stats:  zeroStats,
		}

		// FIXME: stupid
		if needMerge {
			network.TryMergeStats(&origApp, stats)
			continue
		}
		// end

		if v, ok := stats[app.Domain()]; ok {
			node.Stats = v
		}

		network.Nodes = append(network.Nodes, node)
		for _, upstreamID := range app.UpstreamIDs {
			network.Edges = append(network.Edges, VisEdge{
				From: app.ID,
				To:   upstreamID,
			})
		}
	}

	return network, nil
}
