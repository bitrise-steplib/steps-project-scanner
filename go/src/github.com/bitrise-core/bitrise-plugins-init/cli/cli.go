package cli

import (
	"fmt"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-core/bitrise-plugins-init/version"
	"github.com/codegangsta/cli"
)

//=======================================
// Variables
//=======================================

const (
	pluginInputPayloadKey        = "BITRISE_PLUGIN_INPUT_PAYLOAD"
	pluginInputBitriseVersionKey = "BITRISE_PLUGIN_INPUT_BITRISE_VERSION"
	pluginInputTriggerEventKey   = "BITRISE_PLUGIN_INPUT_TRIGGER"
	pluginInputPluginModeKey     = "BITRISE_PLUGIN_INPUT_PLUGIN_MODE"
	pluginInputDataDirKey        = "BITRISE_PLUGIN_INPUT_DATA_DIR"

	bitrisePluginOutputEnvKey = "BITRISE_PLUGIN_OUTPUT"
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
	// If log level defined - use it
	logLevelStr := c.String("loglevel")
	if logLevelStr == "" {
		logLevelStr = "info"
	}

	level, err := log.ParseLevel(logLevelStr)
	if err != nil {
		return err
	}
	log.SetLevel(level)

	// bitriseVersion := os.Getenv(pluginInputBitriseVersionKey)

	// log.Debug("")
	// log.Debugf("pluginInputBitriseVersion: %s", bitriseVersion)

	// triggerEvent := os.Getenv(pluginInputTriggerEventKey)

	// log.Debug("")
	// log.Debugf("pluginInputTriggerEvent: %s", triggerEvent)

	// dataDir := os.Getenv(pluginInputDataDirKey)

	// log.Debug("")
	// log.Debugf("pluginInputDataDir: %s", dataDir)

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
	app.Usage = "Bitrise Init plugin"
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
			Usage:  "Generates a bitrise config files in the current directory.",
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
				cli.BoolFlag{
					Name:  "private",
					Usage: "If true it indicates that source repository is private!",
				},
			},
		},
		cli.Command{
			Name:   "step",
			Usage:  "Generates step template files in the current directory.",
			Action: initStep,
			Flags:  []cli.Flag{},
		},
		cli.Command{
			Name:   "plugin",
			Usage:  "Generates plugin template files in the current directory.",
			Action: initPlugin,
			Flags:  []cli.Flag{},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal("Finished with Error:", err)
	}
}
