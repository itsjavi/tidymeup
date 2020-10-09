package main

import (
	"errors"
	"fmt"
	tm "github.com/buger/goterm"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	_, timeErr := time.LoadLocation("UTC")
	Catch(timeErr)

	var app = &cli.App{
		Usage:                  "Media file organizer",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Value:   false,
				Aliases: []string{"d"},
				Usage:   "Do not process anything, just scan the directory and metadata.",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Value:   false,
				Aliases: []string{"q"},
				Usage:   "It won't print anything, unless it's an error.",
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "run",
				Usage:       "Organizes source folder media into a destination folder.",
				Description: "Organizes the image and video files of a folder recursively from source to destination.",
				ArgsUsage:   "source destination",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "dry-run",
						Value:   false,
						Aliases: []string{"d"},
						Usage:   "Do not process anything, just scan the directory and metadata.",
					},
					&cli.BoolFlag{
						Name:    "quiet",
						Value:   false,
						Aliases: []string{"q"},
						Usage:   "It won't print anything, unless it's an error.",
					},
					&cli.UintFlag{
						Name:    "limit",
						Value:   0,
						Aliases: []string{},
						Usage:   "Limit of files to process.",
					},
					&cli.StringFlag{
						Name:    "extensions",
						Value:   "",
						Aliases: []string{"ext"},
						Usage:   "Custom pipe-separated list of file extensions to process, e.g. \"jpg|mp4|mov|docx\".",
					},
					&cli.StringFlag{
						Name:  "type",
						Value: "",
						Usage: "Custom media type to tag the custom extensions with, e.g. \"document\".",
					},
					&cli.BoolFlag{
						Name:    "fix-dates",
						Value:   false,
						Aliases: []string{"f"},
						Usage:   "Fix the file creation date by using the one in the metadata, if available.",
					},
					&cli.BoolFlag{
						Name:  "db-only",
						Value: false,
						Usage: "Only created the DB index, without moving or copying files.",
					},
					&cli.BoolFlag{
						Name:  "thumbnails",
						Value: false,
						Usage: "Create thumbnails for the compatible images and videos.",
					},
					&cli.BoolFlag{
						Name:    "move",
						Value:   false,
						Aliases: []string{"m"},
						Usage:   "Move the files instead of copying them to the destination.",
					},
					&cli.StringFlag{
						Name:  "exclude",
						Value: "",
						Usage: "Custom pipe-separated list of path patterns to exclude, e.g. \"Screenshot\"",
					},
				},
				Action: func(c *cli.Context) error {
					ctx := AppContext{}

					if c.NArg() == 0 {
						return errors.New("Source and destination directory arguments are missing.")
					}
					if c.NArg() < 2 {
						return errors.New("Destination directory argument is missing.")
					}

					ctx.StartTime = time.Now()
					ctx.SrcDir, _ = filepath.Abs(c.Args().Get(0))
					ctx.DestDir, _ = filepath.Abs(c.Args().Get(1))
					ctx.DryRun = c.Bool("dry-run")
					ctx.Limit = c.Int("limit")
					ctx.CustomExtensions = c.String("extensions")
					ctx.CustomMediaType = c.String("type")
					ctx.CustomExclude = c.String("exclude")
					ctx.FixCreationDates = c.Bool("fix-dates")
					ctx.CreateDbOnly = c.Bool("db-only")
					ctx.CreateThumbnails = c.Bool("thumbnails")
					ctx.MoveFiles = c.Bool("move")
					ctx.Quiet = c.Bool("quiet")

					if !IsDir(ctx.SrcDir) {
						return errors.New("Source directory does not exist.")
					}

					if ctx.SrcDir == ctx.DestDir {
						return errors.New("Source and destination directories cannot be the same.")
					}

					stats := AppRunStats{}
					fileMetaChan := make(chan FileMeta)

					go TidyRoutine(ctx, &stats, fileMetaChan)
					for {
						meta, isOk := <-fileMetaChan
						if isOk == false {
							break // channel closed
						}
						PrintAppStats(meta.Path, stats, ctx)
					}

					PrintAppStats("--", stats, ctx)
					fmt.Println()
					PrintLn(tm.Color("Took %s", tm.BLUE), time.Since(ctx.StartTime))

					return nil
				},
			},
			{
				Name:      "rescan",
				Usage:     "Scans the given mediatidy-generated directory for missing / not imported files and updates the metadata db.",
				ArgsUsage: "dir",
				Action: func(c *cli.Context) error {
					targetDir := c.Args().First()

					if targetDir == "" || !IsDir(targetDir) {
						return errors.New("The given directory does not exist or it is not a directory.")
					}

					ctx := AppContext{SrcDir: targetDir, DestDir: targetDir}
					ctx.StartTime = time.Now()
					ctx.DryRun = c.Bool("dry-run")
					ctx.Quiet = c.Bool("quiet")

					return nil
				},
			},
			{
				Name: "fixdb",
				Action: func(c *cli.Context) error {
					targetDir := c.Args().First()

					if targetDir == "" || !IsDir(targetDir) {
						return errors.New("The given directory does not exist or it is not a directory.")
					}

					ctx := AppContext{SrcDir: targetDir, DestDir: targetDir}
					ctx.InitDb()
					result := ctx.Db.db.Exec("UPDATE files SET path = REPLACE(path, '/.','.')")
					Catch(result.Error)
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	Catch(err)
}
