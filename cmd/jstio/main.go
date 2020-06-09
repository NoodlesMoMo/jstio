package main

import (
	"os"
	"path"

	"git.sogou-inc.com/iweb/jstio/compass"
	"git.sogou-inc.com/iweb/jstio/internel"
	"git.sogou-inc.com/iweb/jstio/internel/logs"
	"github.com/urfave/cli"
)

func main() {
	var (
		region     string
		configPath string
	)

	app := cli.NewApp()
	app.Name = "jstio"
	app.Usage = "A XDS protocol implementor for sogou-ime"
	app.Version = `v0.1.0`

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "region,r",
			Usage:       "jstio run region. product region: [north, south], develop region: [develop]",
			Destination: &region,
			Value:       `develop`,
		},
		cli.StringFlag{
			Name:        "config,c",
			Usage:       "config file path",
			Destination: &configPath,
		},
	}

	app.Before = func(ctx *cli.Context) error {
		if configPath == "" {
			pwd, _ := os.Getwd()
			configPath = path.Join(pwd, "conf/jstio.conf")
		}
		return nil
	}

	app.Action = func(ctx *cli.Context) error {
		if _, err := internel.MustLoadRegionOptions(configPath, region); err != nil {
			return err
		}

		logs.MustInitialization(internel.GetAfxOption().LogPath)

		defer internel.CoreDump()
		return compass.NewCompass().Run()
	}

	if err := app.Run(os.Args); err != nil {
		logs.Logger.WithError(err).Errorln("Oops: jstio crash")
	}
}
