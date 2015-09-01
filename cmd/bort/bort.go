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

// configuration, initialized to defaults
var cfg = Config{
	Nick:       "bort",
	Server:     "irc.freenode.net:6667",
	Address:    bort.DefaultAddress,
	CmdPrefix:  "bort:",
	PollPeriod: 5,
}

// Config holds the configurable values for the program.
type Config struct {
	Nick       string
	Server     string
	Channel    string
	Address    string
	CmdPrefix  string
	PollPeriod uint
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
	con.AddCallback("001", func(*irc.Event) { con.Join(cfg.Channel) })
	con.AddCallback("JOIN", handleJoin)
	con.AddCallback("PART", handleEvent)
	con.AddCallback("PRIVMSG", handleEvent)
	con.AddCallback("CTCP_ACTION", handleEvent)
	con.Loop()
}

// handleJoin waits for bort to join, then connects to bortplug and hands join
// events to handleEvent.
func handleJoin(evt *irc.Event) {
	if evt.Nick != cfg.Nick {
		return
	}
	con.ClearCallback("JOIN")
	con.AddCallback("JOIN", handleEvent)
	log.Printf("joined %s\n", evt.Message())
	connectPlug()
	go pollPushes()
}

// handleEvent processes an incoming message, passes it to bort plug, and
// distributes the results.
func handleEvent(evt *irc.Event) {
	in := evtToMsg(evt)
	msgs := []bort.Message{}
	if client == nil && connectPlug() != nil {
		return
	}
	if err := client.Call("Plugin.Process", in, &msgs); err != nil {
		switch err {
		case rpc.ErrShutdown, io.EOF, io.ErrUnexpectedEOF:
			log.Println("disconnected from bortplug")
			connectPlug()
		default:
			log.Println(err)
		}
		return
	}
	for i := range msgs {
		if err := send(&msgs[i]); err != nil {
			log.Println(err)
		}
	}
}

// pollPlugins periodically fetches and handles messages pushed by plugins.
func deliverPushes() {
	if client == nil {
		if connectPlug() != nil {
			return
		}
	}
	msgs := []bort.Message{}
	if err := client.Call("Plugin.Pull", struct{}{}, &msgs); err != nil {
		switch err {
		case rpc.ErrShutdown, io.EOF, io.ErrUnexpectedEOF:
			log.Println("disconnected from bortplug")
			connectPlug()
		default:
			log.Println(err)
		}
		return
	}
	for i := range msgs {
		if msgs[i].Target == "" {
			msgs[i].Target = cfg.Channel
		}
		if err := send(&msgs[i]); err != nil {
			log.Println(err)
		}
	}
}

// pollPushes periodically delivers pending push messages.
func pollPushes() {
	t := time.Tick(time.Duration(cfg.PollPeriod) * time.Second)
	for {
		<-t
		deliverPushes()
	}
}

// send sends an IRC message according to its content.
func send(msg *bort.Message) error {
	switch msg.Type {
	case bort.None:
	case bort.PrivMsg:
		text := strings.TrimRight(msg.Text, "\n")
		for _, str := range strings.Split(text, "\n") {
			con.Privmsg(msg.Target, str)
		}
	case bort.Action:
		text := msg.Text + "\n" // append newline to ensure SplitN returns 2 strings
		con.Action(msg.Target, strings.SplitN(text, "\n", 2)[0])
	default:
		return fmt.Errorf("unknown message type: %d", msg.Type)
	}
	return nil
}

// evtToMsg converts a go-ircevent Event to a Message.
func evtToMsg(evt *irc.Event) *bort.Message {
	msg := &bort.Message{
		Target:  cfg.Channel,
		Text:    evt.Message(),
		Channel: cfg.Channel,
		Code:    evt.Code,
		Raw:     evt.Raw,
		Nick:    evt.Nick,
		Host:    evt.Host,
		Source:  evt.Source,
		User:    evt.User,
	}
	switch evt.Code {
	case "PRIVMSG":
		msg.Type = bort.PrivMsg
		text := strings.TrimSpace(msg.Text)
		if strings.HasPrefix(text, cfg.CmdPrefix) {
			cmdStr := strings.TrimLeft(text[len(cfg.CmdPrefix):], " ")
			cmdStr += " " // append space to ensure SplitN returns 2 strings
			cmdAndArgs := strings.SplitN(cmdStr, " ", 2)
			if len(cmdAndArgs) == 2 {
				msg.Command = cmdAndArgs[0]
				msg.Args = strings.TrimSpace(cmdAndArgs[1])
			}
		}
	case "CTCP_ACTION":
		msg.Type = bort.Action
	case "JOIN":
		msg.Type = bort.Join
		msg.Text = evt.Nick
	case "PART":
		msg.Type = bort.Part
		msg.Text = evt.Nick
	}
	if evt.Arguments[0] != cfg.Channel {
		msg.Target = evt.Nick
	}
	return msg
}

// connectPlug connects to bortplug's RPC socket.
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

// config overrides defaults with config file and flag values.
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
			cfg.CmdPrefix = flags.CmdPrefix
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
	flag.StringVar(&flags.CmdPrefix, "p", cfg.CmdPrefix, "command prefix")
	flag.UintVar(&flags.PollPeriod, "t", cfg.PollPeriod, "plugin push message poll period in seconds")
	flag.StringVar(&cfgFile, "f", "", "configuration file")
}
