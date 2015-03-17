// Package heckle is a bort IRC bot plugin that responds to a set of watch
// regular expressions with provided retorts.
//
// heckle looks for a pair at the top level of the bort configuration file
// whose key is "heckle" and value is an object that consists of watch/retort
// pairs.
package heckle

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/ianremmler/bort"
)

var (
	retorts = retortMap{}
)

type retortMap map[string]string

func responder(retort string) bort.HandleFunc {
	return func(in, out *bort.Message) error {
		out.Type = bort.PrivMsg
		out.Text = strings.Replace(retort, "%m", in.Match, -1)
		return nil
	}
}

func setup(cfg []byte) {
	if err := json.Unmarshal(cfg, &struct {
		retortMap `json:"heckle"`
	}{retorts}); err != nil {
		log.Println(err)
		return
	}
	for watch, retort := range retorts {
		if _, err := bort.RegisterMatcher(bort.PrivMsg, watch, responder(retort)); err != nil {
			log.Println(err)
		}
	}
}

func init() {
	bort.RegisterSetup(setup)
}
