package es

import (
	"fmt"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/options"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/pkg/util"
	"os"
	"testing"
	"time"
)

func init() {
	dir, _ := os.Getwd()
	cfg := dir + "/../../conf/logserver.yaml"
	_, _ = options.MustLoadRegionOptions(cfg, "develop")
}

func TestESAdapter_Index(t *testing.T) {
	esAdapter := GetElasticAdapter()
	fmt.Println(">>>>:", esAdapter.CurrentIndex())
}

func TestESAdapter_CalcQueryIndexes(t *testing.T) {
	now := time.Now()

	d, _ := time.ParseDuration("-9h")
	from := now.Add(d)
	to := now

	fmt.Println("from:", from, "to:", to)
	GetElasticAdapter().CalcQueryIndexes(from, to)
}

func TestPath(t *testing.T) {
	urls := []string{
		"http://www.xxx.com",
		"http://www.xxx.com?hello&world",
		"http://www.xxx.com?hello?world",
		"?hello=1&world=2",
		"??hello=1&world=2",
	}

	for _, u := range urls {
		fmt.Println(util.ParseBasePath(u))
	}
}
