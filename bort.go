// Package bort provides the base for an IRC bot with plugin capability.
//
// The bot consists of the bort command, which handles the IRC connection, and
// the bortplug command, which handles plugins.  The bortplug command can be
// stopped, recompiled with different or reconfigured plugins, and restarted
// while the bort command stays commected to the IRC server.
//
// Plugins may implement commands, respond to matched text, or push messages
// asynchronously.  Plugins are compiled into the bortplug command.  To enable
// a plugin, add 'import _ "plugin_import_path"' to cmd/bortplug/plugins.go.
//
// Bort looks for a JSON configuration file in ~/.config/bort/bort.conf, which
// can be overridden with a command line parameter.  Bort prioritizes command
// line parameter values, followed by configuration file, and finally, default
// values.  Plugins have access to the configuration file data, and may look
// for values of an appropriate key.
package bort

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"

	"github.com/hashicorp/hcl"
	hclhcl "github.com/hashicorp/hcl/hcl"
)

const (
	// default address for bort/bortplug communication
	DefaultAddress = ":8075"
	cfgFilename    = "bort.conf"
)

var (
	hclCfg         *hclhcl.Object
	defaultCfgFile string
)

// MessageType is a bitmapped IRC message type.
type MessageType int

// message types
const (
	None    MessageType = iota
	PrivMsg MessageType = 1 << iota
	Action
	Join
	Part
	All MessageType = 1<<iota - 1
)

// Message contains all data needed to deal with incoming and outgoing IRC
// messages.
type Message struct {
	// incoming and outgoing
	Type    MessageType
	Context string
	Text    string
	// ignored for outgoing
	Nick    string
	User    string
	Host    string
	IRCCmd  string
	Params  []string
	Command string
	Args    string
	Match   string
}

// HandleFunc provides an interface for handling IRC messages.
type HandleFunc func(in, out *Message) error

// LoadConfig loads the given or default config file
func LoadConfig(cfg interface{}, cfgFile string) error {
	if cfgFile == "" {
		cfgFile = defaultCfgFile
	}
	cfgData, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return err
	}
	if hclCfg, err = hcl.Parse(string(cfgData)); err != nil {
		return fmt.Errorf("%s: %s", cfgFile, err)
	}
	return hcl.DecodeObject(cfg, hclCfg)
}

func init() {
	usr, err := user.Current()
	if err != nil {
		log.Println("error determining home directory")
		return
	}
	defaultCfgFile = filepath.Join(usr.HomeDir, ".config", "bort", cfgFilename)
}
