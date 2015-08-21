// Package urltitle is a bort IRC bot plugin that extracts titles for URLs
// posted in a channel
package urltitle

import (
	"log"
	"net/http"
	"regexp"

	"github.com/ianremmler/bort"
	"github.com/mvdan/xurls"
	"golang.org/x/net/html"
)

var (
	urlRE *regexp.Regexp
)

func findNode(path []string, node *html.Node) *html.Node {
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
	url := urlRE.FindString(in.Text)
	if url == "" {
		return nil
	}
	resp, err := http.Get(url)
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
	title := findNode([]string{"html", "head", "title"}, page)
	if title != nil && title.FirstChild != nil {
		out.Type = bort.PrivMsg
		out.Text = title.FirstChild.Data
	}
	return nil
}

func init() {
	var err error
	if urlRE, err = xurls.StrictMatchingScheme("http"); err != nil {
		log.Println("urltitle: error setting regexp")
		return
	}
	if _, err = bort.RegisterMatcher(bort.PrivMsg, urlRE.String(), extractTitle); err != nil {
		log.Println("urltitle: error registering plugin")
	}
}
