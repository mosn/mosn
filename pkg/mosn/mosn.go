package main

import (
	"errors"
	_ "flag"
	"fmt"
	"github.com/urfave/cli"
	"os"
	"time"
)

func main() {
	app := cli.NewApp()
	app.Name = "mosn"
	app.Version = "0.0.1"
	app.Compiled = time.Now()
	app.Copyright = "(c) 2018 Ant Financial"
	app.Usage = "MOSN is modular observable smart netstub."

	//commands
	app.Commands = []cli.Command{
		cmdStart,
		cmdStop,
		cmdReload,
	}

	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Fprintf(c.App.Writer, "Thar be no %q here.\n", command)
	}
	app.OnUsageError = func(c *cli.Context, err error, isSubcommand bool) error {
		if isSubcommand {
			return err
		}

		fmt.Fprintf(c.App.Writer, "WRONG: %#v\n", err)
		return nil
	}

	//action
	app.Action = func(c *cli.Context) error {
		cli.DefaultAppComplete(c)
		cli.HandleExitCoder(errors.New("not an exit coder, though"))
		cli.ShowAppHelp(c)
		cli.ShowCommandCompletions(c, "nope")
		cli.ShowCommandHelp(c, "also-nope")
		cli.ShowCompletions(c)
		cli.ShowSubcommandHelp(c)
		cli.ShowVersion(c)

		for _, category := range c.App.Categories() {
			fmt.Fprintf(c.App.Writer, "%s\n", category.Name)
			fmt.Fprintf(c.App.Writer, "%#v\n", category.Commands)
			fmt.Fprintf(c.App.Writer, "%#v\n", category.VisibleCommands())
		}

		c.App.Setup()
		return nil
	}

	// ignore error so we don't exit non-zero and break gfmrun README example tests
	_ = app.Run(os.Args)
}
