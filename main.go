package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli"
)

// Version to show in the help text and the --version flag. It is not set
// directly in the sourcecode but set at complite time with
// go build -ldflags "-X main.Version=1.0.0
var Version = "development"

func main() {
	app := cli.NewApp()
	app.Name = "mailimage"
	app.Usage = "An image bord where images are posted via mail"
	//app.HideHelp = true
	app.Version = Version
	//app.ArgsUsage = " " // If it is an empty string, then it shows a stupid default text

	app.Commands = []cli.Command{
		{
			Name:    "insert",
			Aliases: []string{"i"},
			Usage:   "Read an mail from stdin, parse it and save the image into te database",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:   "debug, d",
					Usage:  "debug mode where mails are printed to stdout",
					EnvVar: "DEBUG",
				},
			},
			Action: func(c *cli.Context) error {
				debug = c.Bool("debug")
				insert(os.Stdin)
				return nil
			},
		},
		{
			Name:    "serve",
			Aliases: []string{"s"},
			Usage:   "Serve the images via a webserver",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "listen, l",
					Value: ":5000",
					Usage: "Host and port to listen on",
				},
			},
			Action: func(c *cli.Context) error {
				serve(c.String("listen"))
				return nil
			},
		},
		{
			Name:    "delete",
			Aliases: []string{"d"},
			Usage:   "delete an image by id from the database",
			Action: func(c *cli.Context) error {
				idS := c.Args().First()
				if idS == "" {
					fmt.Printf("No id given")
					os.Exit(1)
				}
				id, err := strconv.Atoi(idS)
				if err != nil {
					fmt.Printf("Id has to be a string")
					os.Exit(1)
				}
				err = deleteFromID(id)
				if err != nil {
					fmt.Printf("Can not delete image: %s", err)
				}

				return nil
			},
		},
	}
	app.Run(os.Args)
}
