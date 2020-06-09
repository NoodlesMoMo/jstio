package options

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

var (
	_afxOption *RegionOptions
)

type RegionType = string

type RegionOptions struct {
	DiskLogRoot         string `yaml:"disk_log_root"`
	ElasticSearchURL    string `yaml:"elastic_search_url"`
	ElasticSearchPrefix string `yaml:"elastic_index_prefix"`
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

func GetOption() *RegionOptions {
	return _afxOption
}
