package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ianremmler/bort/calc"
	"github.com/ianremmler/bort/flip"
	"github.com/ianremmler/bort/forecast"
	"github.com/thoj/go-ircevent"
)

const (
	cmdPrefix = "!"
	table     = "┻━┻"
)

var (
	nick    string
	server  string
	channel string
	con     *irc.Connection
)

func main() {
	lg := log.New(os.Stderr, "bort: ", 0)

	flag.Usage = func() {
		fmt.Println("usage: bort [-n <nick>] [-s <server>] #channel")
	}
	flag.StringVar(&nick, "n", "bort", "nick of the bot")
	flag.StringVar(&server, "s", "irc.freenode.net:6667", "IRC server")
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(0)
	}
	channel = flag.Arg(0)
	if !strings.HasPrefix(channel, "#") {
		lg.Fatalf("%s is not a valid channel", channel)
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
	con.AddCallback("PRIVMSG", handlePrivMsg)
	con.Loop()
}

func handlePrivMsg(evt *irc.Event) {
	text := strings.TrimSpace(evt.Message())
	if !strings.HasPrefix(text, cmdPrefix) {
		return
	}
	cmdStr := text[1:] + " " // append space to ensure splitN returns 2 strings
	cmdAndArgs := strings.SplitN(cmdStr, " ", 2)
	if len(cmdAndArgs) < 2 { // shouldn't happen
		return
	}
	cmd, args := cmdAndArgs[0], strings.TrimSpace(cmdAndArgs[1])

	targ := channel
	if evt.Arguments[0] != channel {
		targ = evt.Nick
	}
	switch cmd {
	case "flip":
		flipped := ""
		if len(args) > 0 {
			flipped = flip.Flip(args)
		} else {
			flipped = table
		}
		con.Privmsg(targ, "(ノಠ益ಠ)ノ彡 "+flipped)
	case "forecast":
		fc, err := forecast.Forecast(args)
		if err != nil {
			con.Privmsg(targ, err.Error())
			return
		}
		for _, line := range strings.Split(fc, "\n") {
			con.Privmsg(targ, line)
		}
	case "calc":
		ans, err := calc.Calc(args, targ == channel)
		if err != nil {
			con.Privmsg(targ, err.Error())
			return
		}
		con.Privmsg(targ, ans)
	}
}
