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

const (
	cmdPrefix = "!"
)

var (
	nick     string
	server   string
	plugAddr string
	channel  string
	con      *irc.Connection
	client   *rpc.Client
	lg       = log.New(os.Stderr, "bort: ", log.Ldate|log.Ltime)
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
		fmt.Println("usage: bort [-n <nick>] [-s <server>] [-p <addr>] #channel")
	}
	flag.StringVar(&nick, "n", "bort", "nick of the bot")
	flag.StringVar(&server, "s", "irc.freenode.net:6667", "IRC server")
	flag.StringVar(&plugAddr, "p", ":1234", "bortplug address")
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
	msg := bort.NewMessage(channel, evt)
	resp := bort.NewResponse()
	if client == nil {
		if connectPlug() != nil {
			return
		}
	}
	if err := client.Call("Event.Process", msg, resp); err != nil {
		lg.Println(err)
		switch err {
		case rpc.ErrShutdown, io.EOF, io.ErrUnexpectedEOF:
			connectPlug()
		}
		return
	}
	switch resp.Type {
	case bort.None:
	case bort.PrivMsg:
		for _, str := range strings.Split(strings.TrimRight(resp.Text, "\n"), "\n") {
			con.Privmsg(resp.Target, str)
		}
	case bort.Action:
		con.Action(resp.Target, strings.SplitN(resp.Text, "\n", 2)[0])
	default:
		lg.Println("error: unknown response type")
	}
}
