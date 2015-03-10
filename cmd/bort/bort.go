package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"strings"

	"github.com/ianremmler/bort"
	"github.com/thoj/go-ircevent"
)

var (
	nick      string
	server    string
	plugAddr  string
	channel   string
	cmdPrefix string
	con       *irc.Connection
	client    *rpc.Client
	lg        = log.New(os.Stderr, "bort: ", log.Ldate|log.Ltime)
)

func connectPlug() error {
	var err error
	if client != nil {
		client.Close()
	}
	client, err = rpc.Dial("tcp", plugAddr)
	if err == nil {
		lg.Println("connected to bortplug")
	} else {
		lg.Println(err)
	}
	return err
}

func init() {
	flag.Usage = func() {
		fmt.Println("usage: bort [-n <nick>] [-s <server>] [-p <addr>] [-c <prefix>] #channel")
	}
	flag.StringVar(&nick, "n", "bort", "nick of the bot")
	flag.StringVar(&server, "s", "irc.freenode.net:6667", "IRC server")
	flag.StringVar(&plugAddr, "p", ":1234", "bortplug address")
	flag.StringVar(&cmdPrefix, "c", "bort:", "command prefix")
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(0)
	}
	channel = flag.Arg(0)
	if !strings.HasPrefix(channel, "#") {
		lg.Fatalf("error: %s is not a valid channel\n", channel)
	}

	con = irc.IRC(nick, nick)
	con.Log = lg
	err := con.Connect(server)
	if err != nil {
		lg.Fatalln(err)
	}

	con.AddCallback("001", func(e *irc.Event) {
		con.Join(channel)
	})
	con.AddCallback("JOIN", func(e *irc.Event) {
		if e.Nick == nick {
			lg.Println("joined", e.Message())
		}
	})
	con.AddCallback("PRIVMSG", handleEvent)

	connectPlug()
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
	if err := client.Call("Event.Process", msg, res); err != nil {
		lg.Println(err)
		switch err {
		case rpc.ErrShutdown, io.EOF, io.ErrUnexpectedEOF:
			connectPlug()
		}
		return
	}
	switch res.Type {
	case bort.None:
	case bort.PrivMsg:
		for _, str := range strings.Split(strings.TrimRight(res.Text, "\n"), "\n") {
			con.Privmsg(res.Target, str)
		}
	case bort.Action:
		con.Action(res.Target, strings.SplitN(res.Text, "\n", 2)[0])
	default:
		lg.Println("error: unknown response type")
	}
}

func newMessage(evt *irc.Event) *bort.Message {
	msg := &bort.Message{
		Channel: channel,
		Host:    evt.Host,
		Nick:    evt.Nick,
		Raw:     evt.Raw,
		Source:  evt.Source,
		Target:  channel,
		Text:    evt.Message(),
		User:    evt.User,
	}
	text := strings.TrimSpace(evt.Message())
	if strings.HasPrefix(text, cmdPrefix) {
		cmdStr := strings.TrimLeft(text[len(cmdPrefix):], " ")
		cmdStr += " " // append space to ensure SplitN returns 2 strings
		cmdAndArgs := strings.SplitN(cmdStr, " ", 2)
		if len(cmdAndArgs) == 2 {
			msg.Command = cmdAndArgs[0]
			msg.Text = strings.TrimSpace(cmdAndArgs[1])
		}
	}
	if evt.Arguments[0] != channel {
		msg.Target = evt.Nick
	}
	return msg
}
