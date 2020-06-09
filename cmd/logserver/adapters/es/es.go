package es

import (
	"context"
	"encoding/json"
	"errors"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/options"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/pkg/adapter"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/pkg/diskany"
	"git.sogou-inc.com/iweb/jstio/cmd/logserver/pkg/util"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	alg "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v2"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"math"
	"os"
	"sync"
	"time"
)

var (
	_esAdapter *ESAdapter
	_esOnce    = sync.Once{}
)

const (
	ElasticTimeFormat = `2006-01-02 15:04:05` //yyyy-MM-dd HH:mm:ss
	OverFlowLogPath   = `/search/odin/accesslog_err/2006-01-02/es-overflow.15.log`
)

type ElasticTime time.Time

func (et *ElasticTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(*et).Format(ElasticTimeFormat) + `"`), nil
}

type LogProperties struct {
	App                  string     `json:"app"`
	Pod                  string     `json:"pod"`
	OdinCluster          string     `json:"oc"`
	Domain               string     `json:"domain"`
	Level                string     `json:"level"`
	FileName             string     `json:"file_name"`
	RequestID            string     `json:"req_id"`
	UpstreamAddr         string     `json:"upstream_addr"`
	UpstreamCluster      string     `json:"upstream_cluster"`
	UpstreamFailedReason string     `json:"upstream_fail_reason"`
	StarTime             *time.Time `json:"star_time"`
	Elapsed              float64    `json:"elapsed"`
	ProtocolVersion      string     `json:"protocol"`
	RequestMethod        string     `json:"req_method"`
	Authority            string     `json:"authority"`
	UA                   string     `json:"ua"`
	BasePath             string     `json:"base_path"`
	Path                 string     `json:"path"`
	ReqHeaderLen         uint64     `json:"req_header_len"`
	RespCode             uint32     `json:"resp_code"`
	RespHeaderLen        uint64     `json:"resp_header_len"`
	RespBodyLen          uint64     `json:"resp_body_len"`
	RespCodeDetail       string     `json:"resp_code_detail"`
	RespFlags            string     `json:"resp_flags"`
}

type ESAdapter struct {
	cli        *elastic.Client
	logger     *logrus.Logger
	diskAny    *diskany.DiskLogServer
	logEntries chan *LogProperties
}

func NewESAdapter() *ESAdapter {
	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	//cli, err := elastic.NewClient(elastic.SetURL(options.GetOption().ElasticSearchURL), elastic.SetErrorLog(logger),elastic.SetTraceLog(logger))
	cli, err := elastic.NewClient(elastic.SetURL(options.GetOption().ElasticSearchURL), elastic.SetErrorLog(logger))
	if err != nil {
		panic(err)
	}

	inst := &ESAdapter{
		cli:        cli,
		logger:     logger,
		diskAny:    diskany.GetDiskLogger(),
		logEntries: make(chan *LogProperties, 8192),
	}
	inst.autoServe()

	return inst
}

func (esa *ESAdapter) CurrentIndex() string {
	today := time.Now().Format("2006-01-02")
	return options.GetOption().ElasticSearchPrefix + today
}

func (esa *ESAdapter) CalcQueryIndexes(from, to time.Time) []string {
	const dayFormat = `2006-01-02`

	indexes := make([]string, 0)

	prefix := options.GetOption().ElasticSearchPrefix

	indexes = append(indexes, prefix+from.Format(dayFormat))

	days := int(math.Round(to.Sub(from).Hours() / 24))
	if days < 0 {
		return nil
	}

	dayDuration, _ := time.ParseDuration("24h")

	for i := 1; i < days+1; i++ {
		indexes = append(indexes, prefix+from.Add(dayDuration*time.Duration(i)).Format(dayFormat))
	}

	return indexes
}

func (esa *ESAdapter) Search(from, to time.Time) (result map[string]interface{}, err error) {
	const (
		domainAggName   = `group_by_domain`
		levelAggName    = `group_by_level`
		basePathAggName = `group_by_path`
	)
	result = make(map[string]interface{})

	indexes := esa.CalcQueryIndexes(from, to)
	if len(indexes) == 0 {
		err = errors.New("invalid range time param")
		return
	}

	pathAgg := elastic.NewTermsAggregation().Field("base_path").Size(3)
	levelAgg := elastic.NewTermsAggregation().Field("level").SubAggregation(basePathAggName, pathAgg)
	domainAgg := elastic.NewTermsAggregation().Field("domain").SubAggregation(levelAggName, levelAgg)
	rangeQuery := elastic.NewBoolQuery().Filter(elastic.NewRangeQuery("star_time").From(from).To(to))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//resp, err := esa.cli.Search().CurrentIndex(esa.CurrentIndex()).Query(rangeQuery).Aggregation(domainAggName, domainAgg).Do(ctx)

	resp, err := esa.cli.Search(indexes...).Query(rangeQuery).Aggregation(domainAggName, domainAgg).Do(ctx)
	if err != nil {
		return
	}

	domains, ok := resp.Aggregations.Terms(domainAggName)
	if !ok {
		err = errors.New("aggregation by domain error")
		return
	}
	for _, bucket := range domains.Buckets {
		item := map[string]interface{}{
			"_total": bucket.DocCount,
		}

		levels, ok := bucket.Terms(levelAggName)
		if !ok {
			err = errors.New("aggregation by level error")
			return
		}

		for _, levelBucket := range levels.Buckets {
			subItem := map[string]interface{}{
				"_total": levelBucket.DocCount,
			}
			paths, ok := levelBucket.Terms(basePathAggName)
			if !ok {
				continue
			}
			for _, pathBucket := range paths.Buckets {
				if key, ok := pathBucket.Key.(string); ok && key != "" {
					subItem[key] = pathBucket.DocCount
				}
			}

			if key, ok := levelBucket.Key.(string); ok && key != "" {
				item[key] = subItem
			}

		}
		if key, ok := bucket.Key.(string); ok && key != "" {
			result[key] = item
		}
	}

	return
}

func (esa *ESAdapter) Sync(meta *adapter.MetaData, entry *alg.HTTPAccessLogEntry) error {

	commonProperties := entry.CommonProperties
	upstreamAddr := ""
	if addr, ok := commonProperties.UpstreamRemoteAddress.Address.(*envoy_api_v2_core.Address_SocketAddress); ok {
		upstreamAddr = addr.SocketAddress.Address
	}
	accessTime := time.Unix(commonProperties.StartTime.Seconds, int64(commonProperties.StartTime.Nanos))
	elapsed := float64(commonProperties.TimeToLastUpstreamRxByte.Seconds) + float64(commonProperties.TimeToLastUpstreamRxByte.Nanos)/1000000000
	lp := LogProperties{
		Level:                meta.Level,
		App:                  meta.App,
		Pod:                  meta.Pod,
		OdinCluster:          meta.OdinCluster,
		Domain:               meta.Domain,
		FileName:             meta.FileName,
		RequestID:            entry.Request.RequestId,
		UpstreamAddr:         upstreamAddr,
		UpstreamCluster:      commonProperties.UpstreamCluster,
		UpstreamFailedReason: commonProperties.UpstreamTransportFailureReason,
		StarTime:             &accessTime,
		Elapsed:              elapsed,
		ProtocolVersion:      entry.ProtocolVersion.String(),
		RequestMethod:        entry.Request.RequestMethod.String(),
		Authority:            entry.Request.Authority,
		UA:                   entry.Request.UserAgent,
		BasePath:             util.ParseBasePath(entry.Request.Path),
		Path:                 entry.Request.Path,
		ReqHeaderLen:         entry.Request.RequestHeadersBytes,
		RespCode:             entry.Response.ResponseCode.Value,
		RespFlags:            commonProperties.ResponseFlags.String(),
		RespHeaderLen:        entry.Response.ResponseHeadersBytes,
		RespBodyLen:          entry.Response.ResponseBodyBytes,
		RespCodeDetail:       entry.Response.ResponseCodeDetails,
	}

	select {
	case esa.logEntries <- &lp:
	default:
		name := accessTime.Format(OverFlowLogPath)
		line, _ := json.Marshal(&lp)
		_ = esa.diskAny.WriteBytes(name, line)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := esa.cli.Index().Index(esa.CurrentIndex()).BodyJson(&lp).Do(ctx)
	if err != nil {
		esa.logger.WithError(err).Errorln("sync to elastic error:", err.Error())
		return err
	}

	return nil
}

func (esa *ESAdapter) indexLog(entry *LogProperties) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := esa.cli.Index().Index(esa.CurrentIndex()).BodyJson(entry).Do(ctx)
	if err != nil {
		esa.logger.WithError(err).Errorln("sync to elastic error:", err.Error())
		return err
	}
	return nil
}

func (esa *ESAdapter) autoServe() {
	go func() {
		for entry := range esa.logEntries {
			_ = esa.indexLog(entry)
		}
	}()
}

func GetElasticAdapter() *ESAdapter {
	if _esAdapter != nil {
		return _esAdapter
	}

	_esOnce.Do(func() {
		_esAdapter = NewESAdapter()
	})

	return _esAdapter
}
