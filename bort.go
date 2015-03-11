package bort

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"text/tabwriter"
)

var (
	setupFuncs     = []func(cfgData []byte){}
	commands       = map[string]*command{}
	matchers       = []*matcher{}
	help           string
	configData     []byte
	defaultCfgFile = "bort.conf"
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

func SetupPlugins() {
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

func LoadConfig(cfgFile string) ([]byte, error) {
	if cfgFile == "" {
		cfgFile = defaultCfgFile
	}
	cfgData, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		configData = cfgData
	}
	return configData, err
}

func init() {
	if usr, err := user.Current(); err == nil {
		defaultCfgFile = filepath.Join(usr.HomeDir, ".config", "bort", defaultCfgFile)
	}
}
