package flip

import (
	"strings"

	"github.com/ianremmler/bort"
)

const (
	table   = "┻━┻"
	flipper = "(ノಠ益ಠ)ノ彡 "
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

func init() {
	for k, v := range flipTable {
		flipTable[v] = k
	}
	bort.RegisterCommand("flip", "flip text (or tables by default)", Flip)
}

func Flip(msg *bort.Message, res *bort.Response) error {
	flipped := ""
	if len(msg.Text) > 0 {
		flipped = flip(msg.Text)
	} else {
		flipped = table
	}
	res.Text = flipper + flipped
	return nil
}

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
