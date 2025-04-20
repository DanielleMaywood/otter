package transformer

import (
	"iter"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type StringCaser struct {
	initialisms map[string]string
	titleCaser  cases.Caser
}

func NewStringCaser(initialisms map[string]string) StringCaser {
	if initialisms == nil {
		initialisms = map[string]string{}
	}

	return StringCaser{
		initialisms: initialisms,
		titleCaser:  cases.Title(language.English),
	}
}

func (c StringCaser) ToPascalCase(name string) string {
	sb := strings.Builder{}

	for p := range strings.SplitSeq(name, "_") {
		if initialism, found := c.initialisms[p]; found {
			sb.WriteString(initialism)
		} else {
			sb.WriteString(c.titleCaser.String(p))
		}
	}

	return sb.String()
}

func (c StringCaser) ToCamelCase(name string) string {
	sb := strings.Builder{}

	next, stop := iter.Pull(strings.SplitSeq(name, "_"))
	defer stop()

	p, ok := next()
	if !ok {
		return sb.String()
	}

	sb.WriteString(p)

	for {
		p, ok := next()
		if !ok {
			break
		}

		if initialism, found := c.initialisms[p]; found {
			sb.WriteString(initialism)
		} else {
			sb.WriteString(c.titleCaser.String(p))
		}

	}

	return sb.String()
}
