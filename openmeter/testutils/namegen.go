package testutils

import (
	"strings"
	"time"

	"github.com/forscht/namegen"
	"github.com/forscht/namegen/dictionaries"
)

var NameGenerator = newNameGenerator()

type nameGenerator struct {
	*namegen.Generator
}

func newNameGenerator() *nameGenerator {
	return &nameGenerator{
		Generator: namegen.New().
			WithSeed(time.Now().UnixNano()).
			WithDictionaries(dictionaries.Adjectives, dictionaries.Animals).
			WithNumberOfWords(2),
	}
}

type GeneratedName struct {
	Key  string
	Name string
}

func (g *nameGenerator) Generate() GeneratedName {
	name := g.Generator.WithStyle(namegen.Title).WithWordSeparator(" ").Generate()

	return GeneratedName{
		Key:  strings.ReplaceAll(strings.ToLower(name), " ", "-"),
		Name: name,
	}
}
