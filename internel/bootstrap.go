package internel

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

var (
	_afxOption *RegionOptions
)

const (
	NorthRegion RegionType = `north`
	SouthRegion RegionType = `south`
	LocalRegion RegionType = `develop`
)

type RegionType = string

type RegionOptions struct {
	DebugMode bool `yaml:"debug_mode"`

	XdsManagerName   string `yaml:"xds_manager_name"`
	XdsManagerListen string `yaml:"xds_manager_listen"`

	DashboardListen string `yaml:"dashboard_listen"`

	Metrics struct {
		Listen string `yaml:"listen"`
		URI    string `yaml:"uri"`
	}

	StatsBackend string `yaml:"stats_backend"`

	LogPath string `yaml:"log_path"`

	ETCDConfig struct {
		Addresses  []string `yaml:"addresses"`
		PrefixKeys []string `yaml:"prefix_keys"`
	} `yaml:"etcd"`

	NSQTopic          string `yaml:"nsq_topic"`
	NSQLookupdAddress string `yaml:"nsqlookupd"`

	MySQLConn struct {
		Addr     string `yaml:"addr"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DB       string `yaml:"db"`
	} `yaml:"mysql_conn"`
}

func (r RegionOptions) GetETCDPrefixKeys() []string {
	return r.ETCDConfig.PrefixKeys
}

func (r RegionOptions) GetETCDEndpoints() []string {
	return r.ETCDConfig.Addresses
}

type AfxOptions map[RegionType]RegionOptions

func MustLoadRegionOptions(cfgName string, region RegionType) (*RegionOptions, error) {
	options := AfxOptions{}
	content, err := ioutil.ReadFile(cfgName)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(content, &options)
	if err != nil {
		panic(err)
	}

	option, ok := options[region]
	if !ok {
		panic(errors.New("unknown region"))
	}
	_afxOption = &option
	return &option, nil
}

func GetAfxOption() *RegionOptions {
	return _afxOption
}
