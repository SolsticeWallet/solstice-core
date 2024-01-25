package utils

import (
	"strings"

	"github.com/tyler-smith/go-bip39/wordlists"
)

func Bip39SuggestWords(v string) []string {
	var res []string
	for _, word := range wordlists.English {
		if strings.HasPrefix(word, v) {
			res = append(res, word)
		}
	}
	return res
}
