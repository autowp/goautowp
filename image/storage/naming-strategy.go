package storage

import (
	"fmt"
	"github.com/fiam/gounidecode/unidecode"
	"regexp"
	"strconv"
	"strings"
)

const (
	NamingStrategyTypeSerial  string = "serial"
	NamingStrategyTypePattern string = "pattern"
)

type GenerateOptions struct {
	Pattern       string
	Extension     string
	Index         int
	Count         int
	PreferredName string
}

type NamingStrategy interface {
	Generate(GenerateOptions) string
}

const ItemPerDir = 1000

type NamingStrategySerial struct {
	deep int
}

type NamingStrategyPattern struct {
}

var specialCharacters = map[rune]string{
	'№':  "N",
	' ':  "_",
	'"':  "_",
	'/':  "_",
	'*':  "_",
	'`':  "_",
	'#':  "_",
	'&':  "_",
	'\'': "_",
	'!':  "_",
	'@':  "_",
	'$':  "s",
	'%':  "_",
	'^':  "_",
	'=':  "-",
	'|':  "_",
	'?':  "_",
	'„':  ",",
	'“':  "_",
	'”':  "_",
	'{':  "(",
	'}':  ")",
	':':  "-",
	';':  "_",
	'-':  "-",
}

func replaceSpecialCharacters(s string) string {
	str := ""
	for _, c := range s {
		d, ok := specialCharacters[c]
		if ok {
			str += d
		} else {
			str += string(c)
		}
	}
	return str
}

func safeFilename(filename string) string {
	filename = unidecode.Unidecode(filename)

	filename = strings.ToLower(filename)

	filename = replaceSpecialCharacters(filename)

	re := regexp.MustCompile("[^A-Za-z0-9.(){}_-]")
	filename = re.ReplaceAllString(filename, "_")

	filename = strings.Trim(filename, "_-")

	re2 := regexp.MustCompile("[_]{2,}")
	filename = re2.ReplaceAllString(filename, "_")

	switch filename {
	case ".", "..", "":
		filename = "_"
	}

	return filename
}

func normalizePattern(pattern string) string {
	a := regexp.MustCompile("[/\\\\]+")
	patternComponents := a.Split(pattern, -1)

	result := make([]string, 0)

	for _, component := range patternComponents {
		switch component {
		case ".", "..":

		default:
			result = append(result, safeFilename(component))
		}
	}

	return strings.Join(result, "/")
}

func (s NamingStrategyPattern) Generate(options GenerateOptions) string {
	pattern := normalizePattern(options.Pattern)
	components := make([]string, 0)
	if len(pattern) > 0 {
		components = append(components, pattern)
	}

	if options.Index > 0 || len(pattern) <= 0 {
		components = append(components, strconv.Itoa(options.Index))
	}

	result := strings.Join(components, "")

	if len(options.Extension) > 0 {
		result = result + "." + options.Extension
	}

	return result
}

func (s NamingStrategySerial) Generate(options GenerateOptions) string {

	fileIndex := options.Count + 1

	dirPath := path(fileIndex, s.deep)

	fileBasename := strconv.Itoa(fileIndex)
	if len(options.PreferredName) > 0 {
		fileBasename = safeFilename(options.PreferredName)
	}

	suffix := ""
	if options.Index > 0 {
		suffix = "_" + strconv.Itoa(fileIndex)
	}

	result := fileBasename + suffix
	if len(options.Extension) > 0 {
		result = result + "." + options.Extension
	}

	return dirPath + result
}

func path(index int, deep int) string {
	chars := len(strconv.Itoa(ItemPerDir - 1)) // use log10, fkn n00b
	path := ""
	if deep > 0 {
		cur := index / ItemPerDir
		for i := 0; i < deep; i++ {
			div := cur / ItemPerDir
			mod := cur - div*ItemPerDir
			path = fmt.Sprintf("%0"+strconv.Itoa(chars)+"d", mod) + "/" + path
			cur = div
		}
	}
	return path
}
