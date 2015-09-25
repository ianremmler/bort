package main

import (
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

// configuration, initialized to defaults
var cfg = Config{
	Address:    bort.DefaultAddress,
	OutboxSize: 10,
}

// Config holds the configurable values for the program.
type Config struct {
	Address    string
	OutboxSize uint
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
		log.Printf("connected to bort (%s)\n", cfg.Address)
		rpc.ServeConn(con)
		log.Println("disconnected from bort")
	}
}

// config overrides defaults with config file and flag values.
func config() {
	if err := bort.LoadConfig(&cfg, cfgFile); err != nil {
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
	flag.StringVar(&cfgFile, "f", "", "configuration file")
}
