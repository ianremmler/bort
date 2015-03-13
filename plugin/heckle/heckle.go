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
	for key, retort := range retorts {
		if _, err := bort.RegisterMatcher(bort.PrivMsg, key, responder(retort)); err != nil {
			log.Println(err)
		}
	}
}

func init() {
	bort.RegisterSetup(setup)
}
