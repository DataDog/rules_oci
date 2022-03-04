package main

import (
	"github.com/DataDog/rules_oci/internal/flagutil"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var app = &cli.App{
	Name: "ocitool",
	Before: func(c *cli.Context) error {
		log.SetLevel(log.InfoLevel)

		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		return nil
	},
	Commands: []*cli.Command{
		{
			Name:   "pull",
			Usage:  "Pull an OCI artifact",
			Action: PullCmd,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "shallow",
					Usage: "Pull only the top level manifests.",
					Value: false,
				},
			},
		},
		{
			Name:   "generate-build-files",
			Action: GenerateBuildFilesCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "image-digest",
				},
			},
		},
		{
			Name:   "create-layer",
			Action: CreateLayerCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "dir",
					Required: true,
				},
				&cli.StringSliceFlag{
					Name: "file",
				},
				&cli.StringFlag{
					Name: "out",
				},
				&cli.StringFlag{
					Name: "outd",
				},
				&cli.GenericFlag{
					Name:  "symlink",
					Value: &flagutil.KeyValueFlag{},
				},
			},
		},
		{
			Name:   "append-layers",
			Action: AppendLayersCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "base",
					Required: true,
				},
				&cli.StringFlag{
                    Name: "bazel-version-file",
                },
                &cli.StringSliceFlag{
					Name: "layer",
				},
				&cli.StringFlag{
					Name: "outd",
				},
				&cli.StringFlag{
					Name: "os",
				},
				&cli.StringFlag{
					Name: "arch",
				},
				&cli.GenericFlag{
					Name:  "annotations",
					Value: &flagutil.KeyValueFlag{},
				},
				&cli.StringFlag{
					Name: "out-manifest",
				},
				&cli.StringFlag{
					Name: "out-config",
				},
				&cli.StringFlag{
					Name: "out-layout",
				},
			},
		},
		{
			Name:   "push",
			Action: PushCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "layout-relative",
				},
				&cli.StringFlag{
					Name: "desc",
				},
				&cli.StringFlag{
					Name: "target-ref",
				},
			},
		},
		{
			Name:   "digest",
			Action: DigestCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "desc",
				},
				&cli.StringFlag{
					Name: "out",
				},
			},
		},
		{
			Name:   "create-index",
			Action: CreateIndexCmd,
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name: "desc",
				},
				&cli.GenericFlag{
					Name:  "annotations",
					Value: &flagutil.KeyValueFlag{},
				},
				&cli.StringFlag{
					Name: "out-index",
				},
				&cli.StringFlag{
					Name: "out-layout",
				},
				&cli.StringFlag{
					Name: "outd",
				},
			},
		},
		{
			Name:   "create-image-manifest",
			Action: CreateImageManifestCmd,
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name: "config-desc",
				},
				&cli.StringSliceFlag{
					Name: "layer-desc",
				},
				&cli.StringFlag{
					Name: "out-manifest",
				},
				&cli.StringFlag{
					Name: "out-layout",
				},
				&cli.StringFlag{
					Name: "outd",
				},
            },
		},
		{
			Name:   "push-blob",
			Hidden: true,
			Action: PushBlobCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "ref",
				},
				&cli.StringFlag{
					Name: "file",
				},
			},
		},
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Value: false,
		},
		&cli.StringSliceFlag{
			Name:     "layout",
			Usage:    "Filepath to a directory with the OCI Layout structure, if it doesn't exist it creates a new directory",
			Required: false,
		},
		&cli.UintFlag{
			Name:  "parallel",
			Usage: "Parallelism of pushing/pulling operations",
			Value: 1, // TODO raise, used by pull impl
		},
	},
}

func main() {
	app.RunAndExitOnError()
}
