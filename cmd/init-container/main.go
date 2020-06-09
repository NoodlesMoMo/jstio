package main

import (
	"errors"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

const (
	defaultInputName          = `template/envoy.yaml.template`
	defaultOutputPath         = `/etc/envoy`
	defaultEnvironment        = `product`
	defaultRegion             = `NORTH`
	defaultJstioNorthAddress  = `jstio.shouji.sogou`
	defaultJstioSouthAddress  = `kstio.shouji.sogou`
	defaultJstioTestAddress   = `10.153.59.136`
	defaultXLogNorthAddress   = `xdslog.north.shouji.sogou`
	defaultXLogSouthAddress   = `xdslog.south.shouji.sogou`
	defaultMetricNorthAddress = `10.160.19.220` // FIXME:
	defaultMetricSouthAddress = `10.160.19.220` // FIXME:
	defaultJstioPort          = 8080
	defaultEnvoyAdminPort     = 9901
	defaultLogServerPort      = 9981
	defaultMetricServerPort   = 9991
	defaultJaegerZipkinPort   = 9411
)

var (
	defaultOdinClusterNamespaceTable = map[string]string{
		"test":   "oneclass",
		"venus":  "planet",
		"saturn": "planet",
	}
)

var (
	inputPath  string
	outputPath string
	version    = `2019-10-08`
)

type EnvoyMetaData struct {
	NodeID           string `yaml:"node_id"`
	AppName          string `yaml:"app_name"`
	OdinCluster      string `yaml:"odin_cluster"`
	Region           string `yaml:"region"`
	Namespace        string `yaml:"namespace"`
	Environment      string `yaml:"environment"`
	XdsManagerAddr   string `yaml:"xds_manager_addr"`
	XdsLogServerAddr string `yaml:"xds_logserver_addr"`
	MetricServerAddr string `yaml:"metric_server_addr"`
	AddrType         string `yaml:"addr_type"`
	LogAddrType      string `yaml:"log_addr_type"`
	MetricsAddrType  string `yaml:"metric_addr_type"`
	XdsPort          int    `yaml:"xds_manager_port"`
	AdminPort        int    `yaml:"envoy_admin_port"`
	LogServerPort    int    `yaml:"logserver_port"`
	MetricServerPort int    `yaml:"metric_server_port"`
	JaegerZipkinPort int    `yaml:"zipkin_port"`
}

// defaultAppName: api-ios-blue-69f5d84b4c-jzxhg -> api-ios
func defaultAppName() string {
	hostName, _ := os.Hostname()
	seps := strings.Split(hostName, "-")
	if len(seps) > 2 {
		seps = seps[:len(seps)-2]
	}

	name := strings.Join(seps, "-")

	return name
}

func LocalIP() string {
	conn, err := net.DialTimeout("udp", "www.sogou.com:80", 3*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()

	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return ""
	}

	return host
}

func addrType(addr string) string {
	if net.ParseIP(addr) == nil {
		return `STRICT_DNS`
	}
	return `STATIC`
}

func main() {
	var (
		xdsManagerAddr, xdsLogSrvAddr, metricSrvAddr string
	)

	meta := EnvoyMetaData{}

	app := cli.NewApp()
	app.Name = "pod-init"
	app.Usage = "init pod command-line for jstio"
	app.Version = version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "app, a",
			Usage:       "application name",
			Destination: &meta.AppName,
			Required:    true,
			EnvVar:      "JSTIO_APPNAME",
		},

		cli.StringFlag{
			Name:        "cluster, c",
			Usage:       "odin cluster",
			Destination: &meta.OdinCluster,
			Required:    true,
			EnvVar:      "JSTIO_CLUSTER",
		},

		cli.StringFlag{
			Name:        "namespace, n",
			Usage:       "odin namespace",
			Destination: &meta.Namespace,
			EnvVar:      "JSTIO_NAMESPACE",
		},

		cli.StringFlag{
			Name:        "host, x",
			Usage:       "xds manager server host",
			Destination: &xdsManagerAddr,
			EnvVar:      "JSTIO_ADDR",
		},

		cli.StringFlag{
			Name:        "als",
			Usage:       "xds logserver address",
			Destination: &xdsLogSrvAddr,
			EnvVar:      "JSTIO_LOGSERVER",
		},

		cli.StringFlag{
			Name:        "ms",
			Usage:       "metrics server address",
			Destination: &metricSrvAddr,
			EnvVar:      "JSTIO_METRIC_SERVER",
		},

		cli.StringFlag{
			Name:        "input, i",
			Usage:       "config abs path",
			Destination: &inputPath,
			EnvVar:      "JSTIO_INPUT",
			Value:       defaultInputName,
		},

		cli.StringFlag{
			Name:        "output, o",
			Usage:       "config output path",
			Destination: &outputPath,
			EnvVar:      "JSTIO_OUTPUT",
			Value:       defaultOutputPath,
		},

		cli.StringFlag{
			Name:        "environment, e",
			Usage:       "app run environment",
			Destination: &meta.Environment,
			EnvVar:      "JSTIO_ENV",
			Value:       defaultEnvironment,
		},

		cli.StringFlag{
			Name:        "region, r",
			Usage:       "where node?",
			Destination: &meta.Region,
			EnvVar:      "JSTIO_REGION",
			Value:       defaultRegion,
		},

		cli.IntFlag{
			Name:        "port, p",
			Usage:       "xds manager server port",
			Destination: &meta.XdsPort,
			EnvVar:      "JSTIO_PORT",
			Value:       defaultJstioPort,
		},

		cli.IntFlag{
			Name:        "admin_port, ap",
			Usage:       "envoy admin port",
			Destination: &meta.AdminPort,
			EnvVar:      "JSTIO_ADMIN_PORT",
			Value:       defaultEnvoyAdminPort,
		},

		cli.IntFlag{
			Name:        "log_port,lp",
			Usage:       "logserver port",
			Destination: &meta.LogServerPort,
			EnvVar:      "JSTION_LOGSERVER_PORT",
			Value:       defaultLogServerPort,
		},

		cli.IntFlag{
			Name:        "metric_port,mp",
			Usage:       "metric server port",
			Destination: &meta.MetricServerPort,
			EnvVar:      "JSTION_METRIC_SERVER_PORT",
			Value:       defaultMetricServerPort,
		},

		cli.IntFlag{
			Name:        "zipkin_port, zp",
			Usage:       "jaeger zipkin port",
			Destination: &meta.JaegerZipkinPort,
			EnvVar:      "JSTIO_ZIPKIN_PORT",
			Value:       defaultJaegerZipkinPort,
		},
	}

	app.Before = func(ctx *cli.Context) error {
		hostName, _ := os.Hostname()
		//meta.NodeID = hostName + ":" + LocalIP()
		meta.NodeID = hostName

		if meta.Namespace == "" {
			if dn, ok := defaultOdinClusterNamespaceTable[meta.OdinCluster]; ok {
				meta.Namespace = dn
			} else {
				return errors.New("not define odin namespace")
			}
		}

		if meta.Region = strings.ToUpper(meta.Region); meta.Region == "" {
			meta.Region = "NORTH"
		}

		if xdsManagerAddr != "" {
			meta.XdsManagerAddr = xdsManagerAddr
		} else {
			if meta.Region == "NORTH" {
				if meta.OdinCluster == `test` {
					meta.XdsManagerAddr = defaultJstioTestAddress
				} else {
					meta.XdsManagerAddr = defaultJstioNorthAddress
				}
			} else {
				meta.XdsManagerAddr = defaultJstioSouthAddress
			}
		}

		if xdsLogSrvAddr != "" {
			meta.XdsLogServerAddr = xdsLogSrvAddr
		} else {
			if meta.Region == "NORTH" {
				meta.XdsLogServerAddr = defaultXLogNorthAddress
			} else {
				meta.XdsLogServerAddr = defaultXLogSouthAddress
			}
		}

		if metricSrvAddr != "" {
			meta.MetricServerAddr = metricSrvAddr
		} else {
			if meta.Region == "NORTH" {
				meta.MetricServerAddr = defaultMetricNorthAddress
			} else {
				meta.MetricServerAddr = defaultMetricSouthAddress
			}
		}

		meta.AddrType = addrType(meta.XdsManagerAddr)
		meta.LogAddrType = addrType(meta.XdsLogServerAddr)
		meta.MetricsAddrType = addrType(meta.MetricServerAddr)

		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})

		return nil
	}

	app.Action = func(ctx *cli.Context) error {
		if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
			return err
		}
		envoyBootstrapFile := path.Join(outputPath, "envoy.yaml")
		appMetaDataFile := path.Join(outputPath, "app_meta.yaml")
		output, err := os.OpenFile(envoyBootstrapFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer output.Close()

		appMetaFile, err := os.OpenFile(appMetaDataFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer appMetaFile.Close()

		envoyTemplate := template.Must(template.ParseFiles(inputPath))
		if err = envoyTemplate.Execute(output, meta); err != nil {
			return err
		}
		_, _ = output.Seek(0, io.SeekStart)
		content, err := ioutil.ReadAll(output)
		if err != nil {
			return err
		}

		yamlMeta, err := yaml.Marshal(&meta)
		if err != nil {
			logrus.WithError(err).Errorln("marshal yaml file error")
			return err
		}

		_, _ = appMetaFile.Write(yamlMeta)

		logrus.Println("\n" + string(content) + "\n")

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	} else {
		logrus.Println("envoy init container execute success")
	}
}
