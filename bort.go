package bort

import (
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"
)

const (
	cfgFilename    = "bort.conf"
	DefaultAddress = ":8075" // robotspeak for BOTS
)

var (
	configData     []byte
	defaultCfgFile string
)

type MessageType int

const (
	None MessageType = iota
	PrivMsg
	Action
)

type Message struct {
	// incoming and outgoing
	Type   MessageType
	Target string
	Text   string
	// fields below are ignored for outgoing messages
	Command string
	Match   string
	Channel string
	// from irc.Event
	Code   string
	Raw    string
	Nick   string
	Host   string
	Source string
	User   string
}

type HandleFunc func(in, out *Message) error

func LoadConfig(cfgFile string) ([]byte, error) {
	if cfgFile == "" {
		cfgFile = defaultCfgFile
	}
	cfgData, err := ioutil.ReadFile(cfgFile)
	if err == nil {
		configData = cfgData
	}
	return configData, err
}

func init() {
	usr, err := user.Current()
	if err != nil {
		log.Println("error determining home directory")
		return
	}
	defaultCfgFile = filepath.Join(usr.HomeDir, ".config", "bort", cfgFilename)
}
