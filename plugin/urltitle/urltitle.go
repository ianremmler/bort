// Package urltitle is a bort IRC bot plugin that extracts titles for URLs
// posted in a channel
package urltitle

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ianremmler/bort"
	"github.com/mvdan/xurls"
	"golang.org/x/net/html"
)

var cfg = &Config{
	Prefix: "",
	Suffix: "",
}

type Config struct {
	Prefix string
	Suffix string
}

func findNode(node *html.Node, path ...string) *html.Node {
	for i := range path {
		if node == nil {
			return nil
		}
		for node = node.FirstChild; node != nil; node = node.NextSibling {
			if node.Type == html.ElementNode && node.Data == path[i] {
				break
			}
		}
	}
	return node
}

func extractTitle(in, out *bort.Message) error {
	resp, err := http.Get(in.Match)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	page, err := html.Parse(resp.Body)
	if err != nil {
		return nil
	}
	title := findNode(page, "html", "head", "title")
	if title != nil && title.FirstChild != nil {
		out.Type = bort.PrivMsg
		text := strings.TrimSpace(title.FirstChild.Data)
		text = strings.SplitN(text+"\n", "\n", 2)[0] // first line
		out.Text = cfg.Prefix + text + cfg.Suffix
	}
	return nil
}

func setup(cfgData []byte) error {
	return json.Unmarshal(cfgData, &struct {
		*Config `json:"urltitle"`
	}{cfg})
}

func init() {
	bort.RegisterSetup(setup)
	urlRE, err := xurls.StrictMatchingScheme("http")
	if err != nil {
		log.Println("urltitle: error setting regexp")
		return
	}
	pat := "(" + urlRE.String() + ")"
	if _, err = bort.RegisterMatcher(bort.PrivMsg, pat, extractTitle); err != nil {
		log.Println("urltitle: error registering plugin")
	}
}
