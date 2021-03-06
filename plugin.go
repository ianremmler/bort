package bort

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"text/tabwriter"
)

var (
	outbox     chan Message
	setupFuncs = []SetupFunc{}
	commands   = map[string]*command{}
	matchers   = []*matcher{}
	matcherID  uint64
	help       string
)

// SetupFunc provides a means for plugins to initialize themselves
type SetupFunc func() error

// Plugin provides RPC calls for bort to pass messages to bortplug for
// handling, and to retrieve pending push messages.
type Plugin struct{}

// Process inspects and processes an incoming message.
func (p *Plugin) Process(in *Message, msgs *[]Message) error { // rpc
	if in.Command == "help" {
		*msgs = append(*msgs, Message{Type: PrivMsg, Context: in.Nick, Text: help})
		return nil
	}
	if cmd, ok := commands[in.Command]; ok {
		out := Message{Context: in.Context}
		if err := cmd.handle(in, &out); err != nil {
			return err
		}
		if out.Type != None {
			*msgs = append(*msgs, out)
		}
		return nil
	}
	errs := ""
	for _, match := range matchers {
		if match.types&in.Type == 0 {
			continue
		}
		matches := match.re.FindStringSubmatch(in.Text)
		idx := len(matches) - 1
		if idx < 0 {
			continue
		}
		if idx > 1 {
			idx = 1
		}
		in.Match = matches[idx]
		out := Message{Context: in.Context}
		if err := match.handle(in, &out); err != nil {
			errs += fmt.Sprintln(err)
			continue
		}
		if out.Type != None {
			*msgs = append(*msgs, out)
		}
	}
	if errs != "" {
		return errors.New(errs)
	}
	return nil
}

// Pull fetches queued messages pushed by plugins.
func (p *Plugin) Pull(dummy struct{}, msgs *[]Message) error { // rpc
	n := len(outbox)
	for i := 0; i < n; i++ {
		*msgs = append(*msgs, <-outbox)
	}
	return nil
}

// Push enqueues an outgoing message pushed by a plugin.
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
	id     uint64
	types  MessageType
	re     *regexp.Regexp
	handle HandleFunc
}

// RegisterSetup registers a function to be run once bort has connected and
// joined.  Plugins should typically call this from init() to ensure fn called.
func RegisterSetup(fn SetupFunc) {
	setupFuncs = append(setupFuncs, fn)
}

// RegisterCommand registers a command handler for the given name.  help is a
// one line description of the plugin's purpose.
func RegisterCommand(cmd, help string, handle HandleFunc) error {
	if cmd == "" {
		return errors.New("cannot register empty command name")
	}
	if _, ok := commands[cmd]; ok {
		return fmt.Errorf("%s: command already registered", cmd)
	}
	commands[cmd] = &command{help: help, handle: handle}
	return nil
}

// UnregisterCommand unrigesters the command handler for the given name, if
// found, and returns whether a handler was removed.
func UnregisterCommand(cmd string) bool {
	_, ok := commands[cmd]
	if ok {
		delete(commands, cmd)
	}
	return ok
}

// RegisterMatcher registers a match handler for the given regular expression.
// types is a bitmask that specifies which message types to consider.  The text
// matched (or that of the first capturing group, if any) will be placed in the
// Match field of the message passed to handle.
func RegisterMatcher(types MessageType, match string, handle HandleFunc) (uint64, error) {
	re, err := regexp.Compile(match)
	if err != nil {
		return 0, err
	}
	matcherID++
	m := &matcher{id: matcherID, types: types, re: re, handle: handle}
	matchers = append(matchers, m)
	return matcherID, nil
}

// UnregisterMatcher unrigesters the match handler for the given ID, if found,
// and returns whether a handler was removed.
func UnregisterMatcher(id uint64) bool {
	for i := range matchers {
		if matchers[i].id == id {
			matchers = append(matchers[:i], matchers[i+1:]...)
			return true
		}
	}
	return false
}

// PluginInit calls plugin setup functions, sets up the push queue, and
// generates plugin help text.
func PluginInit(outboxSize uint) {
	outbox = make(chan Message, outboxSize)

	for _, fn := range setupFuncs {
		if err := fn(); err != nil {
			log.Println(err)
		}
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
