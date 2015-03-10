package bort

import (
	"bytes"
	"fmt"
	"log"
	"net/rpc"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/thoj/go-ircevent"
)

const (
	DefaultPort = 1234
	cmdPrefix   = "!"
)

var (
	handlers = map[string]*Handler{}
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
		res.Type = PrivMsg
		res.Target = msg.Nick
		res.Text = help
		return nil
	}
	if handler, ok := handlers[msg.Command]; ok {
		return handler.handle(msg, res)
	}
	return nil
}

type Message struct {
	Channel string
	Command string
	Host    string
	Nick    string
	Raw     string
	Source  string
	Target  string
	Text    string
	User    string
}

func NewMessage(channel string, evt *irc.Event) *Message {
	msg := &Message{
		Channel: channel,
		Host:    evt.Host,
		Nick:    evt.Nick,
		Raw:     evt.Raw,
		Source:  evt.Source,
		Target:  channel,
		Command: "*",
		Text:    evt.Message(),
		User:    evt.User,
	}

	text := strings.TrimSpace(evt.Message())
	if strings.HasPrefix(text, cmdPrefix) {
		cmdStr := text[1:] + " " // append space to ensure SplitN returns 2 strings
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

type Response struct {
	Type   ResponseType
	Target string
	Text   string
}

func NewResponse() *Response {
	return &Response{}
}

type HandleFunc func(msg *Message, res *Response) error

type Handler struct {
	handle HandleFunc
	help   string
}

func Register(cmd, help string, handle HandleFunc) error {
	if _, ok := handlers[cmd]; ok {
		return fmt.Errorf("%s: handler already registered", cmd)
	}
	handlers[cmd] = &Handler{help: help, handle: handle}
	return nil
}

func Init() {
	buf := &bytes.Buffer{}
	tabWrite := tabwriter.NewWriter(buf, 2, 0, 1, ' ', 0)
	cmds := sort.StringSlice{}
	for cmd := range handlers {
		cmds = append(cmds, cmd)
	}
	cmds.Sort()
	for _, cmd := range cmds {
		fmt.Fprintf(tabWrite, "%s:\t%s\n", cmd, handlers[cmd].help)
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
