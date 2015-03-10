package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"time"

	"github.com/ianremmler/bort"
)

var (
	lg       = log.New(os.Stderr, "bortplug: ", log.Ldate|log.Ltime)
	event    = &bort.Event{}
	plugAddr string
)

func init() {
	flag.Usage = func() {
		fmt.Println("usage: bortplug [-p <addr>]")
	}
	flag.StringVar(&plugAddr, "p", ":1234", "bortplug address")
}

func main() {
	flag.Parse()
	bort.Init()

	listen, err := net.Listen("tcp", plugAddr)
	if err != nil {
		lg.Fatalln(err)
	}
	for {
		con, err := listen.Accept()
		if err != nil {
			lg.Println(err)
			time.Sleep(1 * time.Second)
			break
		}
		lg.Println("connected to bort")
		rpc.ServeConn(con)
		lg.Println("disconnected from bort")
	}
}
