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

type ResponseType int

const (
	None ResponseType = iota
	PrivMsg
	Action
)

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
