package cmd

import (
	"log"
	"os"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/vaultspec/version"
)

func Run() {
	ui := &cli.ConcurrentUi{
		Ui: &cli.ColoredUi{
			InfoColor:  cli.UiColorGreen,
			ErrorColor: cli.UiColorRed,
			WarnColor:  cli.UiColorYellow,
			Ui: &cli.BasicUi{
				Reader:      os.Stdin,
				Writer:      os.Stdout,
				ErrorWriter: os.Stderr,
			},
		},
	}

	c := cli.NewCLI("vaultspec", version.Version)
	c.Args = os.Args[1:]
	c.HelpWriter = os.Stdout

	c.Commands = map[string]cli.CommandFactory{
		//"test": func() (cli.Command, error) { return &CliStartCmd{Ui: ui}, nil },
		//"scan":     func() (cli.Command, error) { return nil, nil },
		//"init":     func() (cli.Command, error) { return nil, nil },
		//"validate": func() (cli.Command, error) { return nil, nil },
		"version": func() (cli.Command, error) { return &VersionCmd{Ui: ui}, nil },
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}
	os.Exit(exitStatus)
}
