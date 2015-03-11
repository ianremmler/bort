package main

import (
	"encoding/json"
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
	// flags
	address string
	cfgFile string

	lg  = log.New(os.Stderr, "bortplug: ", log.Ldate|log.Ltime)
	cfg Config

	event = &bort.Event{}
)

type Config struct {
	Address string `json:"address"`
}

func main() {
	flag.Parse()
	config()
	bort.SetupPlugins()

	listen, err := net.Listen("tcp", cfg.Address)
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

func config() {
	cfgData, err := bort.LoadConfig(cfgFile)
	if err != nil {
		lg.Printf("could not read config file: %s\n", cfgFile)
	}
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		lg.Println(err)
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.Address = address
		}
	})
}

func init() {
	err := rpc.Register(event)
	if err != nil {
		log.Fatal(err)
	}
	flag.Usage = func() {
		fmt.Println("usage: bortplug [<options>]")
		flag.PrintDefaults()
	}
	flag.StringVar(&address, "a", ":1234", "bortplug address")
	flag.StringVar(&cfgFile, "c", "", "configuration file")
}
