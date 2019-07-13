package main

import (
	"errors"
	"log"
	"net/url"
	"os"
	"strings"

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
			Usage:  "Hana File System Mount Entry Point",
		},
		cli.StringFlag{
			Name:   "base, b",
			EnvVar: "HANA_TENANT_BASE_PATH",
			Usage:  "Hana Tenant Base Path",
			Value:  "/",
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
	base := c.GlobalString("base")

	if len(host) == 0 {
		return errors.New("Must set the hana tenant hostname")
	}

	if len(mountpoint) == 0 {
		parts := strings.SplitN(host, ".", 2)
		if len(parts) == 2 {
			mountpoint = parts[0]
		} else {
			return errors.New("Must set the mount point")
		}
	}

	if !strings.HasPrefix(base, "/") {
		// add prefix
		base = "/" + base
	}

	uri := &url.URL{
		Host: host,
		User: url.UserPassword(user, password),
		Path: base,
	}

	client, err := hana.NewClient(uri)

	if err != nil {
		return err
	}

	fs := fuse.NewFileSystemHost(fs.NewHanaFS(client))

	fs.SetCapReaddirPlus(true)

	fs.Mount(mountpoint, []string{"-d"})

	return err

}
