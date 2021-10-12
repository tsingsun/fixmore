package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/tsingsun/fixmore"
	"github.com/tsingsun/fixmore/apps"
	"github.com/tsingsun/fixmore/apps/hsfile"
	"github.com/tsingsun/fixmore/socket"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

var (
	configFlag = &cli.PathFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Value:   "./etc/app.yaml",
	}
	demo = &cli.Command{
		Name: "demo",
		Flags: []cli.Flag{
			configFlag,
		},
		Action: func(c *cli.Context) error {
			path := c.Path("config")
			srv := socket.New(socket.Configuration(path), socket.UseLogger())
			serivce, err := fixmore.NewFixService(srv.Configuration, apps.NewDemoFix())
			if err != nil {
				return err
			}
			if err := srv.RegisterService(serivce); err != nil {
				return err
			}
			return srv.Start()
		},
	}
	hundsun = &cli.Command{
		Name: "hundsun",
		Flags: []cli.Flag{
			configFlag,
			&cli.BoolFlag{
				Name:  "test",
				Usage: "if test the date will be: 20210929",
			},
		},
		Action: func(c *cli.Context) error {
			path := c.Path("config")
			test := c.Bool("test")
			srv := socket.New(socket.Configuration(path), socket.UseLogger())
			serivce, err := fixmore.NewFixService(srv.Configuration, hsfile.NewHSFix(srv.Configuration))
			if err != nil {
				return err
			}
			if err := srv.RegisterService(serivce); err != nil {
				return err
			}
			//hs dbf api
			if wtdb, err := hsfile.NewWTDB(srv.Configuration.String("hundsun.dbfdir"), test); err != nil {
				panic(err)
			} else {
				defer wtdb.CloseFile()
			}

			return srv.Start()
		},
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "fix-srv"
	app.Usage = "cli command for FIX server"
	app.Version = fixmore.Version
	app.Commands = []*cli.Command{
		demo,
		hundsun,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err.Error())
		os.Exit(1)
	}
}
