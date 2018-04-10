package main

import (
	"errors"
	_"flag"
	"fmt"
	"os"
	"time"
	"gitlab.alipay-inc.com/afe/mosn/pkg/example/sofarpc"
	"github.com/urfave/cli"
)

func init() {
	//cli.AppHelpTemplate += "\nMOSN is modular observable smart netstub.\n"
	//cli.CommandHelpTemplate += "\nYMMV\n"
	//cli.SubcommandHelpTemplate += "\nor something\n"

	//cli.HelpFlag = cli.BoolFlag{Name: "halp"}
	//cli.BashCompletionFlag = cli.BoolFlag{Name: "compgen", Hidden: true}
	//cli.VersionFlag = cli.BoolFlag{Name: "print-version, V"}
	//
	//cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
	//	fmt.Fprintf(w, "best of luck to you\n")
	//}
	//cli.VersionPrinter = func(c *cli.Context) {
	//	fmt.Fprintf(c.App.Writer, "version=%s\n", c.App.Version)
	//}
	//cli.OsExiter = func(c int) {
	//	fmt.Fprintf(cli.ErrWriter, "refusing to exit %d\n", c)
	//}
	//cli.ErrWriter = ioutil.Discard
	//cli.FlagStringer = func(fl cli.Flag) string {
	//	return fmt.Sprintf("\t\t%s", fl.GetName())
	//}
}

//type hexWriter struct{}
//
//func (w *hexWriter) Write(p []byte) (int, error) {
//	for _, b := range p {
//		fmt.Printf("%x", b)
//	}
//	fmt.Printf("\n")
//
//	return len(p), nil
//}
//
//type genericType struct{
//	s string
//}
//
//func (g *genericType) Set(value string) error {
//	g.s = value
//	return nil
//}
//
//func (g *genericType) String() string {
//	return g.s
//}

func main() {
	app := cli.NewApp()
	app.Name = "mosn"
	app.Version = "0.0.1"
	app.Compiled = time.Now()
	app.Copyright = "(c) 2018 Ant Financial"
	app.Usage = "MOSN is modular observable smart netstub."

	//commands
	app.Commands = []cli.Command{
		//cli.Command{
		//	Name:        "example",
		//	Category:    "motion",
		//	Usage:       "run built-in example",
		//	Subcommands: cli.Commands{
		//		cli.Command{
		//			Name:   "sofarpc",
		//			Action: exampleSofarpcAction,
		//		},
		//	},
		//
		//	Action: func(c *cli.Context) error {
		//		c.Command.FullName()
		//		c.Command.HasName("wop")
		//		c.Command.Names()
		//		c.Command.VisibleFlags()
		//		fmt.Fprintf(c.App.Writer, "please choose one example suite\n")
		//		return nil
		//	},
		//	OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
		//		fmt.Fprintf(c.App.Writer, "usage error\n")
		//		return err
		//	},
		//},
		cmdStart,
		cmdStop,
		cmdReload,
	}

	//flags
	//app.Flags = []cli.Flag{
	//	cli.StringFlag{
	//		Name:  "config, c",
	//		Usage: "Load configuration from `FILE`",
	//		EnvVar: "MOSN_CONFIG",
	//		Value: "config/mson.conf",
	//	},
	//}

	//hooks
	//app.Before = func(c *cli.Context) error {
	//	fmt.Fprintf(c.App.Writer, "HEEEERE GOES\n")
	//	return nil
	//}
	//app.After = func(c *cli.Context) error {
	//	fmt.Fprintf(c.App.Writer, "Phew!\n")
	//	return nil
	//}
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

	//if os.Getenv("HEXY") != "" {
	//	app.Writer = &hexWriter{}
	//	app.ErrWriter = &hexWriter{}
	//}

	//app.Metadata = map[string]interface{}{
	//	"layers":     "many",
	//	"explicable": false,
	//	"whatever-values": 19.99,
	//}


	// ignore error so we don't exit non-zero and break gfmrun README example tests
	_ = app.Run(os.Args)
}

func exampleSofarpcAction(c *cli.Context) error {
	sofarpc.Run()
	return nil
}