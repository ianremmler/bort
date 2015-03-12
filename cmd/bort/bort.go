package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"strings"
	"time"

	"github.com/ianremmler/bort"
	"github.com/thoj/go-ircevent"
)

var (
	// flags
	flags   Config
	cfgFile string

	con    *irc.Connection
	client *rpc.Client
)

var cfg = Config{
	Nick:       "bort",
	Server:     "irc.freenode.net:6667",
	Address:    bort.DefaultAddress,
	Prefix:     "bort:",
	PollPeriod: 5,
}

type Config struct {
	Nick       string `json:"nick"`
	Server     string `json:"server"`
	Channel    string `json:"channel"`
	Address    string `json:"address"`
	Prefix     string `json:"prefix"`
	PollPeriod uint   `json:"poll_period"`
}

func main() {
	flag.Parse()
	config()
	if len(cfg.Channel) < 2 || !strings.HasPrefix(cfg.Channel, "#") {
		log.Fatalf("'%s' is not a valid channel", cfg.Channel)
	}
	if cfg.PollPeriod < 1 {
		cfg.PollPeriod = 1
	}

	con = irc.IRC(cfg.Nick, cfg.Nick)
	if err := con.Connect(cfg.Server); err != nil {
		log.Fatalln(err)
	}
	con.AddCallback("001", func(e *irc.Event) {
		con.Join(cfg.Channel)
	})
	con.AddCallback("JOIN", func(e *irc.Event) {
		if e.Nick == cfg.Nick {
			con.ClearCallback("JOIN")
			log.Printf("joined %s\n", e.Message())
			connectPlug()
			go func() {
				poll := time.Tick(time.Duration(cfg.PollPeriod) * time.Second)
				for {
					<-poll
					deliverPushes()
				}
			}()
		}
	})
	con.AddCallback("PRIVMSG", handleEvent)
	con.Loop()
}

func handleEvent(evt *irc.Event) {
	msg := newMessage(evt)
	res := &bort.Response{}
	if client == nil {
		if connectPlug() != nil {
			return
		}
	}
	if err := client.Call("Plugin.Process", msg, res); err != nil {
		log.Println(err)
		switch err {
		case rpc.ErrShutdown, io.EOF, io.ErrUnexpectedEOF:
			log.Println("disconnected from bortplug")
			connectPlug()
		}
		return
	}
	if err := send(res); err != nil {
		log.Println(err)
	}
}

func deliverPushes() {
	if client == nil {
		if connectPlug() != nil {
			return
		}
	}
	res := []bort.Response{}
	if err := client.Call("Plugin.Pull", struct{}{}, &res); err != nil {
		log.Println(err)
		switch err {
		case rpc.ErrShutdown, io.EOF, io.ErrUnexpectedEOF:
			connectPlug()
		}
		return
	}
	for i := range res {
		if res[i].Target == "" {
			res[i].Target = cfg.Channel
		}
		if err := send(&res[i]); err != nil {
			log.Println(err)
		}
	}
}

func send(res *bort.Response) error {
	switch res.Type {
	case bort.None:
	case bort.PrivMsg:
		for _, str := range strings.Split(strings.TrimRight(res.Text, "\n"), "\n") {
			con.Privmsg(res.Target, str)
		}
	case bort.Action:
		con.Action(res.Target, strings.SplitN(res.Text, "\n", 2)[0])
	default:
		return fmt.Errorf("unknown response type: %d", res.Type)
	}
	return nil
}

func newMessage(evt *irc.Event) *bort.Message {
	msg := &bort.Message{
		Channel: cfg.Channel,
		Host:    evt.Host,
		Nick:    evt.Nick,
		Raw:     evt.Raw,
		Source:  evt.Source,
		Target:  cfg.Channel,
		Text:    evt.Message(),
		User:    evt.User,
	}
	text := strings.TrimSpace(evt.Message())
	if strings.HasPrefix(text, cfg.Prefix) {
		cmdStr := strings.TrimLeft(text[len(cfg.Prefix):], " ")
		cmdStr += " " // append space to ensure SplitN returns 2 strings
		cmdAndArgs := strings.SplitN(cmdStr, " ", 2)
		if len(cmdAndArgs) == 2 {
			msg.Command = cmdAndArgs[0]
			msg.Text = strings.TrimSpace(cmdAndArgs[1])
		}
	}
	if evt.Arguments[0] != cfg.Channel {
		msg.Target = evt.Nick
	}
	return msg
}

func connectPlug() error {
	var err error
	if client != nil {
		client.Close()
	}
	client, err = rpc.Dial("tcp", cfg.Address)
	if err == nil {
		log.Println("connected to bortplug")
	}
	return err
}

func config() {
	if cfgData, err := bort.LoadConfig(cfgFile); err != nil {
		log.Println(err)
	} else if err := json.Unmarshal(cfgData, &cfg); err != nil {
		log.Println(err)
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "n":
			cfg.Nick = flags.Nick
		case "s":
			cfg.Server = flags.Server
		case "c":
			cfg.Channel = flags.Channel
		case "a":
			cfg.Address = flags.Address
		case "p":
			cfg.Prefix = flags.Prefix
		case "t":
			cfg.PollPeriod = flags.PollPeriod
		}
	})
}

func init() {
	flag.StringVar(&flags.Nick, "n", cfg.Nick, "nick of the bot")
	flag.StringVar(&flags.Server, "s", cfg.Server, "IRC server")
	flag.StringVar(&flags.Channel, "c", cfg.Channel, "channel")
	flag.StringVar(&flags.Address, "a", cfg.Address, "bortplug address")
	flag.StringVar(&flags.Prefix, "p", cfg.Prefix, "command prefix")
	flag.UintVar(&flags.PollPeriod, "t", cfg.PollPeriod, "plugin push message poll period in seconds")
	flag.StringVar(&cfgFile, "f", "", "configuration file")
}
