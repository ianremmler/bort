package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"strings"
	"sync"
	"time"

	"github.com/ianremmler/bort"
	"github.com/sorcix/bot"
	"github.com/sorcix/irc"
	"github.com/sorcix/irc/ctcp"
)

var (
	// flags
	flags   Config
	cfgFile string

	mut    sync.Mutex
	isLive bool
	botc   *bot.Client
	rpcc   *rpc.Client
)

// configuration, initialized to defaults
var cfg = &Config{
	Nick:       "bort",
	Server:     "irc.freenode.net:6667",
	Channel:    "#bort",
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
	go pollPushes()
	for {
		run()
	}
}

// run starts the bot
func run() {
	con, err := irc.Dial(cfg.Server)
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second)
		return
	}

	if botc = bot.NewClient(con, handleMessage); botc == nil {
		return
	}
	botc.Identify(cfg.Nick, cfg.Nick, cfg.Nick)
	botc.Wait()

	mut.Lock()
	isLive = false
	mut.Unlock()
}

// setup looks for a welcome response, joins the channel, and connects to bortplug.
func setup(msg *irc.Message, snd irc.Sender) {
	switch msg.Command {
	case irc.RPL_WELCOME:
		log.Printf("connected to IRC server %s (%s)\n", cfg.Server, msg.Name)
		out := &irc.Message{Command: irc.JOIN, Params: []string{cfg.Channel}}
		if err := snd.Send(out); err != nil {
			log.Println(err)
		}
	case irc.JOIN:
		if msg.Name != cfg.Nick {
			break
		}
		if len(msg.Params) > 0 {
			log.Printf("joined %s as %s\n", msg.Params[0], msg.Name)
		}
		isLive = true
	}
}

// handleMessage processes incoming IRC messages.
func handleMessage(msg *irc.Message, snd irc.Sender) {
	if msg == nil || msg.Prefix == nil {
		return
	}

	mut.Lock()
	defer mut.Unlock()

	if !isLive {
		setup(msg, snd)
		return
	}
	if connectPlug() != nil {
		return
	}

	in := convertMsg(msg)
	msgs := []bort.Message{}
	if err := rpcc.Call("Plugin.Process", in, &msgs); err != nil {
		handleRPCError(err)
		return
	}
	sendMessages(msgs)
}

// deliverPushes periodically fetches and handles messages pushed by plugins.
func deliverPushes() {
	mut.Lock()
	defer mut.Unlock()

	if !isLive || connectPlug() != nil {
		return
	}

	msgs := []bort.Message{}
	if err := rpcc.Call("Plugin.Pull", struct{}{}, &msgs); err != nil {
		handleRPCError(err)
		return
	}
	sendMessages(msgs)
}

// handleRPCError handles errors from RPC calls.
func handleRPCError(err error) {
	switch err {
	case nil:
	case rpc.ErrShutdown, io.EOF, io.ErrUnexpectedEOF:
		log.Println("disconnected from bortplug")
		rpcc.Close()
		rpcc = nil
		connectPlug()
	default:
		log.Println(err)
	}
}

// sendMessages sends a slice of messages to the server.
func sendMessages(msgs []bort.Message) {
	for i := range msgs {
		if msgs[i].Context == "" {
			msgs[i].Context = cfg.Channel
		}
		if err := send(botc, &msgs[i]); err != nil {
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

// send sends an IRC message to the server according to its content.
func send(snd irc.Sender, in *bort.Message) error {
	base := irc.Message{Command: irc.PRIVMSG, Params: []string{in.Context}}
	switch in.Type {
	case bort.None:
	case bort.PrivMsg:
		text := strings.TrimRight(in.Text, "\n")
		for _, str := range strings.Split(text, "\n") {
			out := base
			out.Trailing = str
			snd.Send(&out)
		}
	case bort.Action:
		text := strings.SplitN(in.Text+"\n", "\n", 2)[0]
		out := base
		out.Trailing = ctcp.Action(text)
		snd.Send(&out)
	default:
		return fmt.Errorf("unknown message type: %d", in.Type)
	}
	return nil
}

// convertMsg converts an irc.Message to a bort.Message.
func convertMsg(imsg *irc.Message) *bort.Message {
	bmsg := &bort.Message{
		Context: cfg.Channel,
		IRCCmd:  imsg.Command,
		Nick:    imsg.Name,
		User:    imsg.User,
		Host:    imsg.Host,
		Params:  append([]string(nil), imsg.Params...),
		Text:    imsg.Trailing,
	}
	if len(bmsg.Params) > 0 && bmsg.Params[0] != cfg.Channel {
		bmsg.Context = bmsg.Nick
	}
	switch bmsg.IRCCmd {
	case irc.PRIVMSG:
		// handle actions
		if tag, text, ok := ctcp.Decode(bmsg.Text); ok && tag == ctcp.ACTION {
			bmsg.Type = bort.Action
			bmsg.Text = text
			break
		}

		bmsg.Type = bort.PrivMsg
		isCmd := (bmsg.Context != cfg.Channel)
		text := strings.TrimSpace(bmsg.Text)
		if strings.HasPrefix(text, cfg.CmdPrefix) {
			text = strings.TrimLeft(text[len(cfg.CmdPrefix):], " ")
			isCmd = true
		}
		if isCmd {
			cmdAndArgs := strings.SplitN(text+" ", " ", 2)
			if len(cmdAndArgs) == 2 {
				bmsg.Command = cmdAndArgs[0]
				bmsg.Args = strings.TrimSpace(cmdAndArgs[1])
			}
		}
	case irc.JOIN:
		bmsg.Type = bort.Join
		bmsg.Text = bmsg.Nick
	case irc.PART:
		bmsg.Type = bort.Part
		bmsg.Text = bmsg.Nick
	}
	return bmsg
}

// connectPlug connects to bortplug's RPC socket.
func connectPlug() error {
	if rpcc != nil {
		return nil
	}

	var err error
	rpcc, err = rpc.Dial("tcp", cfg.Address)
	if err == nil {
		log.Printf("connected to bortplug (%s)\n", cfg.Address)
	}
	return err
}

// config overrides defaults with config file and flag values.
func config() {
	if err := bort.LoadConfig(cfg, cfgFile); err != nil {
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
