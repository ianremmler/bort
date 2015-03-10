package bort

import (
	"bytes"
	"fmt"
	"log"
	"net/rpc"
	"regexp"
	"sort"
	"text/tabwriter"
)

const (
	DefaultPort = 1234
)

var (
	commands = map[string]*command{}
	matchers = []*matcher{}
	event    = &Event{}
	help     string
)

type ResponseType int

const (
	None ResponseType = iota
	Action
	PrivMsg
)

type Event struct{}

func (e *Event) Process(msg *Message, res *Response) error {
	res.Type = PrivMsg
	res.Target = msg.Target
	if msg.Command == "help" {
		res.Target = msg.Nick
		res.Text = help
		return nil
	}
	if cmd, ok := commands[msg.Command]; ok {
		return cmd.handle(msg, res)
	}
	for _, mtch := range matchers {
		matches := mtch.re.FindStringSubmatch(msg.Text)
		idx := len(matches) - 1
		if idx < 0 {
			continue
		}
		if idx > 1 {
			idx = 1
		}
		msg.Match = matches[idx]
		return mtch.handle(msg, res)
	}
	return nil
}

type Message struct {
	Channel string
	Command string
	Host    string
	Match   string
	Nick    string
	Raw     string
	Source  string
	Target  string
	Text    string
	User    string
}

type Response struct {
	Type   ResponseType
	Target string
	Text   string
}

type HandleFunc func(msg *Message, res *Response) error

type command struct {
	handle HandleFunc
	help   string
}

type matcher struct {
	re     *regexp.Regexp
	handle HandleFunc
}

func RegisterCommand(cmd, help string, handle HandleFunc) error {
	if _, ok := commands[cmd]; ok {
		return fmt.Errorf("%s: command already registered", cmd)
	}
	commands[cmd] = &command{help: help, handle: handle}
	return nil
}

func RegisterMatcher(match string, handle HandleFunc) error {
	re, err := regexp.Compile(match)
	if err != nil {
		return err
	}
	matchers = append(matchers, &matcher{re: re, handle: handle})
	return nil
}

func Init() {
	buf := &bytes.Buffer{}
	tabWrite := tabwriter.NewWriter(buf, 2, 0, 1, ' ', 0)
	cmds := sort.StringSlice{}
	for cmd := range commands {
		cmds = append(cmds, cmd)
	}
	cmds.Sort()
	for _, cmd := range cmds {
		fmt.Fprintf(tabWrite, "%s:\t%s\n", cmd, commands[cmd].help)
	}
	tabWrite.Flush()
	help = buf.String()
}

func init() {
	err := rpc.Register(event)
	if err != nil {
		log.Fatal(err)
	}
}
