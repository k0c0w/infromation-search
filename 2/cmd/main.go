package main

import (
	"iter"
	"log"
	"main/fileReader"
	"main/morph"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type void = struct{}

const indexPath string = "C:\\CustomDesktop\\informations search\\1\\output\\index.json"
const output string = "C:\\CustomDesktop\\informations search\\2\\output"

// https://pymorphy2.readthedocs.io/en/stable/user/grammemes.html
var STOP_TAGS map[string]void = map[string]void{"PREP": {}, "CONJ": {}, "PRCL": {}, "INTJ": {}} // пропускать предлоги, союзы, частицы, междометия
var POS_TAGS map[string]string = map[string]string{
	"NOUN": "_NOUN",
	"VERB": "_VERB", "INFN": "_VERB", "GRND": "_VERB", "PRTF": "_VERB", "PRTS": "_VERB",
	"ADJF": "_ADJ", "ADJS": "_ADJ",
	"ADVB": "_ADV",
	"PRED": "_ADP",
}

var russianLowwerRanges []*unicode.RangeTable = []*unicode.RangeTable{{R16: []unicode.Range16{{0x0430, 0x044f, 1}, {0x0451, 0x0451, 1}, {0x2010, 0x2010, 1}}}}

var STOP_WORDS map[string]void = map[string]void{"и": {}, "в": {}, "во": {}, "не": {}, "что": {}, "он": {}, "на": {}, "я": {}, "с": {}, "со": {}, "как": {}, "а": {}, "то": {}, "все": {}, "она": {}, "так": {}, "его": {}, "но": {}, "да": {}, "ты": {}, "к": {}, "у": {}, "же": {}, "вы": {}, "за": {}, "бы": {}, "по": {}, "только": {}, "ее": {}, "её": {}, "мне": {}, "было": {}, "вот": {}, "от": {}, "меня": {}, "еще": {}, "ещё": {}, "нет": {}, "о": {}, "из": {}, "ему": {}, "теперь": {}, "когда": {}, "даже": {}, "ну": {}, "вдруг": {}, "ли": {}, "если": {}, "уже": {}, "или": {}, "ни": {}, "быть": {}, "был": {}, "него": {}, "до": {}, "вас": {}, "нибудь": {}, "опять": {}, "уж": {}, "вам": {}, "ведь": {}, "там": {}, "потом": {}, "себя": {}, "ничего": {}, "ей": {}, "может": {}, "они": {}, "тут": {}, "где": {}, "есть": {}, "надо": {}, "ней": {}, "для": {}, "мы": {}, "тебя": {}, "их": {}, "чем": {}, "была": {}, "сам": {}, "чтоб": {}, "без": {}, "будто": {}, "чего": {}, "раз": {}, "тоже": {}, "себе": {}, "под": {}, "будет": {}, "ж": {}, "тогда": {}, "кто": {}, "этот": {}, "того": {}, "потому": {}, "этого": {}, "какой": {}, "совсем": {}, "ним": {}, "здесь": {}, "этом": {}, "один": {}, "почти": {}, "мой": {}, "тем": {}, "чтобы": {}, "нее": {}, "сейчас": {}, "были": {}, "куда": {}, "зачем": {}, "всех": {}, "никогда": {}, "можно": {}, "при": {}, "наконец": {}, "два": {}, "об": {}, "другой": {}, "хоть": {}, "после": {}, "над": {}, "больше": {}, "тот": {}, "через": {}, "эти": {}, "нас": {}, "про": {}, "всего": {}, "них": {}, "какая": {}, "много": {}, "разве": {}, "три": {}, "эту": {}, "моя": {}, "впрочем": {}, "хорошо": {}, "свою": {}, "этой": {}, "перед": {}, "иногда": {}, "лучше": {}, "чуть": {}, "том": {}, "нельзя": {}, "такой": {}, "им": {}, "более": {}, "всегда": {}, "конечно": {}, "всю": {}, "между": {}, "ооо": {}}

func trimNonRussian(word string) string {
	runes := []rune(word)
	start := 0
	end := len(runes) - 1

	for start <= end && !unicode.In(runes[start], unicode.Cyrillic) {
		start++
	}

	for end >= start && !unicode.In(runes[end], unicode.Cyrillic) {
		end--
	}

	if start <= end {
		return string(runes[start : end+1])
	}

	return ""
}

func splitSymbols(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r) && r != '-'
}

func lematizationAndFiltering(words iter.Seq[string]) []string {
	result := make([]string, 0)

	for complexWord := range words {
		simplerWords := strings.FieldsFunc(strings.ToLower(complexWord), splitSymbols)
		for _, word := range simplerWords {
			word = trimNonRussian(word)
			if word == "" || len(word) == 1 {
				continue
			}

			_, morphNorms, morphTags := morph.Parse(word)
			if len(morphNorms) == 0 {
				log.Printf("No norms for %s", word)
				result = append(result, word)
				continue
			}

			suffixes := make(map[string]bool)

			for i, tags := range morphTags {
				norm := morphNorms[i]
				tag := strings.Split(tags, ",")[0]
				_, hasStopTag := STOP_TAGS[tag]
				if hasStopTag {
					break
				}

				suffix, hasPosTag := POS_TAGS[tag]
				_, hasSuffix := suffixes[suffix]
				if _, ok := STOP_WORDS[norm]; hasPosTag && !hasSuffix && !ok {
					result = append(result, norm)
					suffixes[suffix] = true
				}
			}
		}

	}

	return result
}

func main() {
	indexFileReader, err := fileReader.New(indexPath)
	if err != nil {
		log.Fatalln(err)
	}

	for fileReader := range indexFileReader.WordReaders() {
		words, err := fileReader.Words()

		if err == nil {
			finalWords := lematizationAndFiltering(words)
			finalText := strings.Join(finalWords, " ")

			path := fileReader.GetFilePath()
			fileName := filepath.Base(path)
			file, err := os.Create(filepath.Join(output, fileName))
			if err != nil {
				log.Fatalln(err)
			}
			defer file.Close()
			file.WriteString(finalText)

		} else {
			log.Println(err)
		}
	}
}
