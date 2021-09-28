package main

import (
	"github.com/tsingsun/fixmore"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "fix"
	app.Usage = "cli command for FIX"
	app.Version = fixmore.Version
	app.Commands = []*cli.Command{
		TradeClientCmd,
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err.Error())
		os.Exit(1)
	}
}
