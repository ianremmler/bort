// Package flip is a bort IRC bot plugin that flips tables and text, in true
// emoji rage style.
package flip

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/ianremmler/bort"
)

const (
	table          = "┻━┻"
	defaultFlipper = "(ノಠ益ಠ)ノ彡 "
)

var flipTable = map[rune]rune{
	'a':  'ɐ',
	'b':  'q',
	'c':  'ɔ',
	'd':  'p',
	'e':  'ǝ',
	'f':  'ɟ',
	'g':  'ƃ',
	'h':  'ɥ',
	'i':  'ı',
	'j':  'ɾ',
	'k':  'ʞ',
	'l':  'ʃ',
	'm':  'ɯ',
	'n':  'u',
	'r':  'ɹ',
	't':  'ʇ',
	'v':  'ʌ',
	'w':  'ʍ',
	'y':  'ʎ',
	'.':  '˙',
	'[':  ']',
	'(':  ')',
	'{':  '}',
	'?':  '¿',
	'!':  '¡',
	'\'': ',',
	'<':  '>',
	'_':  '‾',
	'&':  '⅋',
	';':  '؛',
	'"':  '„',
}

var cfg = &Config{
	Flipper: defaultFlipper,
}

type Config struct {
	Flipper string `json:"flipper"`
}

func init() {
	for k, v := range flipTable {
		flipTable[v] = k
	}
	bort.RegisterSetup(setup)
	bort.RegisterCommand("flip", "flip text (or tables by default)", Flip)
}

// Flip draws the "emoji table flip guy" flipping the given text, or a table if
// no text is provided.
func Flip(in, out *bort.Message) error {
	flipped := ""
	if len(in.Args) > 0 {
		flipped = flip(in.Args)
	} else {
		flipped = table
	}
	out.Type = bort.PrivMsg
	out.Text = cfg.Flipper + flipped
	return nil
}

// flip generates a string that approximates the appearance of the text
// rotated half way using carefully selected unicode characters.
func flip(text string) string {
	out := ""
	for _, char := range strings.ToLower(text) {
		outChar := char
		if flipChar, ok := flipTable[char]; ok {
			outChar = flipChar
		}
		out = string(outChar) + out
	}
	return out
}

func setup(cfgData []byte) {
	if err := json.Unmarshal(cfgData, &struct {
		*Config `json:"flip"`
	}{cfg}); err != nil {
		log.Println(err)
		return
	}
}
