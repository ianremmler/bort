package bort

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"text/tabwriter"
)

var (
	outbox     chan Response
	setupFuncs = []func(cfgData []byte){}
	commands   = map[string]*command{}
	matchers   = []*matcher{}
	help       string
)

type Plugin struct{}

func (p *Plugin) Process(msg *Message, res *Response) error { // rpc
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
	for _, match := range matchers {
		matches := match.re.FindStringSubmatch(msg.Text)
		idx := len(matches) - 1
		if idx < 0 {
			continue
		}
		if idx > 1 {
			idx = 1
		}
		msg.Match = matches[idx]
		return match.handle(msg, res)
	}
	return nil
}

func (p *Plugin) Pull(dummy struct{}, res *[]Response) error { // rpc
	for {
		select {
		case push := <-outbox:
			*res = append(*res, push)
		default:
			return nil
		}
	}
}

func Push(res *Response) error {
	select {
	case outbox <- *res:
		return nil
	default:
		return errors.New("outbox full")
	}
}

type command struct {
	handle HandleFunc
	help   string
}

type matcher struct {
	re     *regexp.Regexp
	handle HandleFunc
}

func RegisterSetup(fn func(cfgData []byte)) {
	setupFuncs = append(setupFuncs, fn)
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

func PluginInit(outboxSize uint) {
	outbox = make(chan Response, outboxSize)

	for _, fn := range setupFuncs {
		fn(configData)
	}
	setupFuncs = nil

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
