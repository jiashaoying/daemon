package main

import (
	"fmt"
	"github.com/jiashaoying/daemon"
	"log"
	"os"
)

var (
	cmd string
)

func init() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], "install|remove|start|stop|status|log")
		return
	}
	cmd = os.Args[1]
}

func main() {
	var d daemon.Daemon
	var err error
	if d, err = daemon.New(
		daemon.WithName("wrapother"),
		daemon.WithExec("/home/shgsec/wrapother"),
		daemon.WithUser("shgsec"),
		daemon.WithGroup("shgsec"),
		daemon.WithLogFile("/home/shgsec/wrapother.log"),
		daemon.WithDescription("test wrapother service"),
		daemon.WithLockFile("/home/shgsec/wrapother.lock"),
		daemon.WithPidFile("/home/shgsec/wrapother.pid"),
	); err != nil {
		log.Panicln(err)
	}

	switch cmd {
	case "install":
		if err := d.Install(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Succeeded")
		}
	case "enable":
		if err := d.Enable(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Succeeded")
		}
	case "disable":
		if err := d.Disable(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Succeeded")
		}
	case "remove":
		if err := d.Remove(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Succeeded")
		}
	case "start":
		if err := d.Start(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Succeeded")
		}
	case "stop":
		if err := d.Stop(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Succeeded")
		}
	case "status":
		if err := d.Status(); err != nil {
			fmt.Println(err)
		}
	case "log":
		if err := d.Log(); err != nil {
			fmt.Println(err)
		}
	default:
		fmt.Println("Usage: ", os.Args[0], "install|enable|disable|remove|start|stop|status|log")
	}
}
