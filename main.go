package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/ostcar/mailimage/internal/mailimage"
	"github.com/urfave/cli"
)

// Version to show in the help text and the --version flag. It is not set
// directly in the sourcecode but set at complite time with
// go build -ldflags "-X main.Version=1.0.0
var version = "development"

const defaultLogPath = "/var/log/mailimage.log"

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	app := cli.NewApp()
	app.Name = "mailimage"
	app.Usage = "An image bord where images are posted via mail"
	app.Version = version

	app.Commands = []cli.Command{
		{
			Name:  "serve",
			Usage: "Serve the images via http",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "listen, l",
					Value: ":5000",
					Usage: "Host and port to listen on",
				},
			},
			Action: func(c *cli.Context) error {
				// Log to file
				if os.Getenv("DEBUG") == "" {
					logPath := os.Getenv("LOG_PATH")
					if logPath == "" {
						logPath = defaultLogPath
					}

					f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
					if err != nil {
						log.Fatalf("Can not open logfile: %s", err)
					}
					defer f.Close()

					log.SetOutput(f)
				}

				return mailimage.Serve(c.String("listen"))
			},
		},
		{
			Name:  "insert",
			Usage: "Read an mail from stdin, parse it and save the image into te database",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:   "debug, d",
					Usage:  "debug mode where mails are printed to stdout",
					EnvVar: "DEBUG",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Bool("debug") {
					os.Setenv("FOO", "1")
				}
				return mailimage.Insert(os.Stdin)
			},
		},
		{
			Name:  "delete",
			Usage: "delete an image by id from the database",
			Action: func(c *cli.Context) error {
				idS := c.Args().First()
				if idS == "" {
					fmt.Printf("No id given\n")
					os.Exit(1)
				}

				id, err := strconv.Atoi(idS)
				if err != nil {
					fmt.Printf("Id has to be a string\n")
					os.Exit(1)
				}

				return mailimage.Delete(id)
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
