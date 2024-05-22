package main

import (
	"github.com/DataDog/rules_oci/go/internal/flagutil"
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
				&cli.PathFlag{
					Name:  "configuration-file",
					Usage: "Path to a configuration file. Useful when there are too many flags to pass at once.",
				},
				&cli.StringFlag{
					Name: "dir",
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
				&cli.StringFlag{
					Name: "bazel-label",
				},
				&cli.GenericFlag{
					Name:  "symlink",
					Value: &flagutil.KeyValueFlag{},
				},
				&cli.GenericFlag{
					Name:  "file-map",
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
				&cli.GenericFlag{
					Name:  "layer",
					Value: &flagutil.KeyValueFlag{},
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
				&cli.GenericFlag{
					Name:  "labels",
					Value: &flagutil.KeyValueFlag{},
				},
				&cli.StringSliceFlag{
					Name: "env",
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
				&cli.StringFlag{
					Name: "entrypoint",
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
				&cli.StringFlag{
					Name: "parent-tag",
				},
				&cli.GenericFlag{
					Name:  "headers",
					Value: &flagutil.KeyValueFlag{},
				},
				&cli.GenericFlag{
					Name:  "x_meta_headers",
					Value: &flagutil.KeyValueFlag{},
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
			Name: "create-oci-image-layout",
			Description: `Creates a directory containing an OCI Image Layout based on the input layout,
as described in https://github.com/opencontainers/image-spec/blob/main/image-layout.md.`,
			Action: CreateOciImageLayoutCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "layout-relative",
				},
				&cli.StringFlag{
					Name: "desc",
				},
				&cli.StringSliceFlag{
					Name:  "base-image-layouts",
					Usage: "A comma separated list of directory paths, each path containing an OCI Image Layout.",
				},
				&cli.StringFlag{
					Name:  "out-dir",
					Usage: "The directory that the OCI Image Layout will be written to.",
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
		{
			Name:   "push-rules",
			Hidden: true,
			Action: PublishRulesCmd,
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
