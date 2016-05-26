package cli

import (
	"fmt"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-init/version"
	"github.com/codegangsta/cli"
)

//=======================================
// Functions
//=======================================

func printVersion(c *cli.Context) {
	fmt.Fprintf(c.App.Writer, "%v\n", c.App.Version)
}

func before(c *cli.Context) error {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		ForceColors:     true,
		TimestampFormat: "15:04:05",
	})

	// Log level
	logLevelStr := c.String("loglevel")
	if logLevelStr == "" {
		logLevelStr = "info"
	}

	level, err := log.ParseLevel(logLevelStr)
	if err != nil {
		return err
	}
	log.SetLevel(level)

	return nil
}

//=======================================
// Main
//=======================================

// Run ...
func Run() {
	// Parse cl
	cli.VersionPrinter = printVersion

	app := cli.NewApp()

	app.Name = path.Base(os.Args[0])
	app.Usage = "Bitrise Init Tool"
	app.Version = version.VERSION

	app.Author = ""
	app.Email = ""

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "loglevel, l",
			Usage:  "Log level (options: debug, info, warn, error, fatal, panic).",
			EnvVar: "LOGLEVEL",
		},
		cli.BoolFlag{
			Name:   "ci",
			Usage:  "If true it indicates that we're used by another tool so don't require any user input!",
			EnvVar: "CI",
		},
	}
	app.Before = before
	app.Commands = []cli.Command{
		cli.Command{
			Name:   "config",
			Usage:  "Generates a bitrise config files based on your project.",
			Action: initConfig,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "dir",
					Usage: "Directory to scan.",
				},
				cli.StringFlag{
					Name:  "output-dir",
					Usage: "Directory to save scan results.",
				},
			},
		},
		cli.Command{
			Name:   "default-config",
			Usage:  "Generates default bitrise config files.",
			Action: manualInitConfig,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "output-dir",
					Usage: "Directory to save scan results.",
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal("Finished with Error:", err)
	}
}
