// Package heckle is a bort IRC bot plugin that responds to a set of watch
// regular expressions with provided retorts.
//
// heckle looks for a pair at the top level of the bort configuration file
// whose key is "heckle" and value is an object that consists of watch/retort
// pairs.
package heckle

import (
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

func setup() error {
	if err := bort.ConfigPlugin("heckle", &retorts); err != nil {
		return err
	}
	for watch, retort := range retorts {
		if _, err := bort.RegisterMatcher(bort.PrivMsg, watch, responder(retort)); err != nil {
			log.Println(err)
		}
	}
	return nil
}

func init() {
	bort.RegisterSetup(setup)
}
