package main

import (
	"github.com/tsingsun/fixmore"
	"github.com/tsingsun/fixmore/socket"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "fix-srv"
	app.Usage = "cli command for FIX server"
	app.Version = fixmore.Version
	app.Action = func(c *cli.Context) error {
		path := c.Path("config")
		srv := socket.New(socket.Configuration(path), socket.UseLogger())
		serivce, err := fixmore.NewFixService(srv.Configuration)
		if err != nil {
			return err
		}
		if err := srv.RegisterService(serivce); err != nil {
			return err
		}
		return srv.Start()
	}
	app.Flags = []cli.Flag{
		&cli.PathFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Value:   "./etc/app.yaml",
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err.Error())
		os.Exit(1)
	}
}
