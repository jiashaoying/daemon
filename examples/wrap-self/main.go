package main

import (
	"fmt"
	"github.com/jiashaoying/daemon"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) > 1 {
		cmd := os.Args[1]
		d, err := daemon.New(
			daemon.WithUser("shgsec"),
			daemon.WithGroup("shgsec"),
			daemon.WithLogFile("/home/shgsec/wrapself.log"),
			daemon.WithDescription("test wrapself service"),
			daemon.WithLockFile("/home/shgsec/wrapself.lock"),
			daemon.WithPidFile("/home/shgsec/wrapself.pid"),
		)
		if err != nil {
			fmt.Println(err)
			return
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
		return
	}

	cp, _ := os.Getwd()
	log.Println("working dir is", cp)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	t := time.NewTicker(3 * time.Second)
Loop:
	for {
		select {
		case <-t.C:
			log.Println("ticking...")
		case <-quit:
			log.Println("Stop tick daemon ...")
			t.Stop()
			break Loop
		}
	}

}
