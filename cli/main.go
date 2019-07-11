package main

import (
	"log"
	"net/url"
	"os"

	"github.com/billziss-gh/cgofuse/fuse"

	"github.com/Soontao/hanafs/fs"

	"github.com/Soontao/hanafs/hana"

	"github.com/urfave/cli"
)

// Version string, in release version
// This variable will be over writted by complier
var Version = "SNAPSHOT"

// AppName of this application
var AppName = "Hana FS"

// AppUsage of this application
var AppUsage = "A Command Line Tool for Hana File System"

func main() {

	flags := []cli.Flag{
		cli.StringFlag{
			Name:   "user, u",
			EnvVar: "HANA_USER",
			Usage:  "Hana User",
		},
		cli.StringFlag{
			Name:   "password, p",
			EnvVar: "HANA_PASSWORD",
			Usage:  "Hana Password",
		},
		cli.StringFlag{
			Name:   "host, h",
			EnvVar: "HANA_TENANT",
			Usage:  "Hana Tenant Hostname",
		},
		cli.StringFlag{
			Name:   "mount, m",
			EnvVar: "MOUNT_PATH",
			Value:  "./hana",
			Usage:  "Hana File System Mount Entry Point",
		},
	}

	app := cli.NewApp()
	app.Version = Version
	app.Name = AppName
	app.Usage = AppUsage
	app.Author = "Theo Sun"
	app.EnableBashCompletion = true
	app.Flags = flags
	app.Action = appAction
	app.HideHelp = true

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func appAction(c *cli.Context) (err error) {
	user := c.GlobalString("user")
	password := c.GlobalString("password")
	host := c.GlobalString("host")
	mountpoint := c.GlobalString("mount")

	uri := &url.URL{
		Host: host,
		User: url.UserPassword(user, password),
	}

	client, err := hana.NewClient(uri)

	if err != nil {
		return err
	}

	fs := fuse.NewFileSystemHost(fs.NewHanaFS(client))

	defer func() {
		fs.Unmount()
	}()

	fs.SetCapReaddirPlus(true)

	fs.Mount(mountpoint, []string{"-d"})

	return err

}
