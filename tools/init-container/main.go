package main

import (
	"errors"
	"github.com/sirupsen/logrus"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli"
)

const (
	defaultInputName      = `template/envoy.yaml.template`
	defaultOutputName     = `/etc/envoy/envoy.yaml`
	defaultEnvironment    = `product`
	defaultRegion         = `beijing`
	defaultJstioAddress   = `jstio.shouji.sogou`
	defaultJstioPort      = 8080
	defaultEnvoyAdminPort = 9901
)

var (
	defaultOdinClusterNamespaceTable = map[string]string{
		"test":   "oneclass",
		"venus":  "planet",
		"saturn": "planet",
	}
)

var (
	inputName  string
	outputName string
	version    = `2019-08-29`
)

type EnvoyMetaData struct {
	NodeID         string
	AppName        string
	OdinCluster    string
	Region         string
	Namespace      string
	Environment    string
	XdsManagerAddr string
	AddrType       string
	XdsPort        int
	AdminPort      int
	// ...
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

	//return strings.TrimSuffix(name, "-blue")
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

func main() {

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
			Name:        "input, i",
			Usage:       "config abs path",
			Destination: &inputName,
			EnvVar:      "JSTIO_INPUT",
			Value:       defaultInputName,
		},

		cli.StringFlag{
			Name:        "output, o",
			Usage:       "config output path",
			Destination: &outputName,
			EnvVar:      "JSTIO_OUTPUT",
			Value:       defaultOutputName,
		},

		cli.StringFlag{
			Name:        "host, x",
			Usage:       "xds manager server host",
			Destination: &meta.XdsManagerAddr,
			EnvVar:      "JSTIO_HOST",
			Value:       defaultJstioAddress,
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
	}

	app.Before = func(ctx *cli.Context) error {
		hostName, _ := os.Hostname()
		//meta.NodeID = hostName + ":" + LocalIP()
		meta.NodeID = hostName

		if net.ParseIP(meta.XdsManagerAddr) == nil {
			meta.AddrType = `STRICT_DNS`
		} else {
			meta.AddrType = `STATIC`
		}

		if meta.Namespace == "" {
			if dn, ok := defaultOdinClusterNamespaceTable[meta.OdinCluster]; ok {
				meta.Namespace = dn
			} else {
				return errors.New("not define odin namespace")
			}
		}

		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})

		return nil
	}

	app.Action = func(ctx *cli.Context) error {
		if err := os.MkdirAll(filepath.Dir(outputName), os.ModePerm); err != nil {
			return err
		}

		output, err := os.OpenFile(outputName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer output.Close()

		envoyTemplate := template.Must(template.ParseFiles(inputName))
		if err = envoyTemplate.Execute(output, meta); err != nil {
			return err
		}
		_, _ = output.Seek(0, io.SeekStart)
		content, err := ioutil.ReadAll(output)
		if err != nil {
			return err
		}

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
