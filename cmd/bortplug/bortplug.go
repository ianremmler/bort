package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/rpc"
	"time"

	"github.com/ianremmler/bort"
)

var (
	// flags
	flags   Config
	cfgFile string

	plug = &bort.Plugin{}
)

var cfg = Config{
	Address:    bort.DefaultAddress,
	OutboxSize: 10,
}

type Config struct {
	Address    string `json:"address"`
	OutboxSize uint   `json:"outbox_size"`
}

func main() {
	if err := rpc.Register(plug); err != nil {
		log.Fatal(err)
	}

	flag.Parse()
	config()

	bort.PluginInit(cfg.OutboxSize)

	listen, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		con, err := listen.Accept()
		if err != nil {
			log.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Println("connected to bort")
		rpc.ServeConn(con)
		log.Println("disconnected from bort")
	}
}

func config() {
	if cfgData, err := bort.LoadConfig(cfgFile); err != nil {
		log.Println(err)
	} else if err := json.Unmarshal(cfgData, &cfg); err != nil {
		log.Println(err)
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.Address = flags.Address
		case "o":
			cfg.OutboxSize = flags.OutboxSize
		}
	})
}

func init() {
	flag.StringVar(&flags.Address, "a", cfg.Address, "bortplug address")
	flag.UintVar(&flags.OutboxSize, "o", cfg.OutboxSize, "outbox size")
	flag.StringVar(&cfgFile, "c", "", "configuration file")
}
