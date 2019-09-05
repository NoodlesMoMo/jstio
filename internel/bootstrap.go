package internel

import (
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"jstio/internel/logs"
	"jstio/internel/util"
	"os"
	"path"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	afxMeta *AfxMetaData
)

func init() {
	afxMeta = mustNewAfxMeta()
}

type DBConn struct {
	Addr     string `yaml:"addr"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DB       string `yaml:"db"`
}

type NSQConn struct {
	Host         string `yaml:"host"`
	NSQDPort     int    `yaml:"nsqd_tcp_port"`
	NSQDHTTPPort int    `yaml:"nsqd_http_port"`
}

type NSQNode struct {
	BroadcastAddress string `json:"broadcast_address"`
	TCPPort          int    `json:"tcp_port"`
	HTTPPort         int    `json:"http_port"`
}

type NSQCluster struct {
	LookupAddress string `yaml:"nsqlookupd_addr"`
	Nodes         []NSQNode
}

type AfxMetaData struct {
	ClusterName      string `yaml:"cluster_name"`
	AccessLog        string `yaml:"access_log"`
	XdsManagerListen string `yaml:"xds_manager_listen"`
	DashboardListen  string `yaml:"dashboard_listen"`
	PushMode         string `yaml:"push_mode"`

	DebugMode bool `yaml:"debug_mode"`

	ETCDEndpoints []string `yaml:"etcd_endpoints"`

	MySQLConn DBConn `yaml:"mysql_conn"`

	NSQCluster NSQCluster `yaml:"nsq_cluster"`

	Node string
}

func (amd *AfxMetaData) String() string {
	return amd.ClusterName
}

func (amd *AfxMetaData) loadMetaData() error {

	pwd, _ := os.Getwd()
	absPathName := path.Join(pwd, "conf/jstio.conf")

	content, err := ioutil.ReadFile(absPathName)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(content, amd)
}

func (amd *AfxMetaData) fetchNSQDEndpoints() error {
	tagLog := logs.FuncTaggedLoggerFactory()

	amd.NSQCluster.Nodes = make([]NSQNode, 0)

	code, resp, err := fasthttp.GetTimeout(nil, amd.NSQCluster.LookupAddress+"/nodes", 3*time.Second)
	if err != nil {
		tagLog("response").Errorln(err)
		return err
	}

	if code != fasthttp.StatusOK {
		tagLog("response").Errorln("code:", code)
	}

	producers := make(map[string][]NSQNode)

	if err = json.Unmarshal(resp, &producers); err != nil {
		tagLog("unmarshal").Errorln(err)
	}

	amd.NSQCluster.Nodes = producers["producers"]
	if amd.NSQCluster.Nodes == nil && len(amd.NSQCluster.Nodes) == 0 {
		tagLog("endpoints").Errorln("no usable endpoint")
	}

	return nil
}

func (amd *AfxMetaData) NSQDEndpoints() (tcpAddr, httpAddr []string) {
	for _, nsq := range amd.NSQCluster.Nodes {
		tcpAddr = append(tcpAddr, fmt.Sprintf("%s:%d", nsq.BroadcastAddress, nsq.TCPPort))
		httpAddr = append(httpAddr, fmt.Sprintf("%s:%d", nsq.BroadcastAddress, nsq.HTTPPort))
	}
	return
}

func mustNewAfxMeta() *AfxMetaData {
	meta := &AfxMetaData{}
	if err := meta.loadMetaData(); err != nil {
		logs.Logger.WithField(`bootstrap`, `load meta data`).Panic(err)
	}

	hostName, _ := os.Hostname()
	meta.Node = hostName + ":" + util.GetLocalIPV4Addr()

	_ = meta.fetchNSQDEndpoints()

	runMode := `product`
	if meta.DebugMode {
		runMode = `debug`
	}
	logs.Logger.WithField(`bootstrap`, `run mode`).Println(runMode)

	return meta
}

func GetAfxMeta() *AfxMetaData {
	return afxMeta
}
