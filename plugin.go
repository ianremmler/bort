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
	outbox     chan Message
	setupFuncs = []func(cfgData []byte){}
	commands   = map[string]*command{}
	matchers   = []*matcher{}
	help       string
)

type Plugin struct{}

func (p *Plugin) Process(in, out *Message) error { // rpc
	out.Type = PrivMsg
	out.Target = in.Target
	if in.Command == "help" {
		out.Target = in.Nick
		out.Text = help
		return nil
	}
	if cmd, ok := commands[in.Command]; ok {
		return cmd.handle(in, out)
	}
	for _, match := range matchers {
		matches := match.re.FindStringSubmatch(in.Text)
		idx := len(matches) - 1
		if idx < 0 {
			continue
		}
		if idx > 1 {
			idx = 1
		}
		in.Match = matches[idx]
		return match.handle(in, out)
	}
	return nil
}

func (p *Plugin) Pull(dummy struct{}, msgs *[]Message) error { // rpc
	for {
		select {
		case msg := <-outbox:
			*msgs = append(*msgs, msg)
		default:
			return nil
		}
	}
}

func Push(msg *Message) error {
	select {
	case outbox <- *msg:
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
	outbox = make(chan Message, outboxSize)

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
