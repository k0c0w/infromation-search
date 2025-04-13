package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const INVERTED_INDEX_PATH path = "C:\\CustomDesktop\\informations search\\3\\output\\inverted_index.json"
const DOCUMENTS_FOLDER path = "C:\\CustomDesktop\\informations search\\2\\output"
const OUTPUT_FOLDER_PATH path = "C:\\CustomDesktop\\informations search\\4\\output"

type path = string
type fileId = string
type word = string

type InvertedIndex map[word][]fileId

type TfIdf struct {
	Tf    map[fileId]map[word]float64
	Idf   map[word]float64
	TfIdf map[fileId]map[word]float64
}

func main() {
	tfIdf := tfIdf(loadInvertedIndex(INVERTED_INDEX_PATH))
	saveResults(tfIdf)
}

func tfIdf(invertedIndex InvertedIndex) TfIdf {
	files := getFilePaths()
	totalFilesCount := len(files)
	uniqueWordsCount := len(invertedIndex)

	tfIdf := make(map[fileId]map[word]float64, uniqueWordsCount)

	tf := tf(uniqueWordsCount, files)
	idf := idf(invertedIndex, float64(totalFilesCount))

	for word, filesContainingWord := range invertedIndex {
		for _, fileId := range filesContainingWord {
			if tfIdf[fileId] == nil {
				tfIdf[fileId] = make(map[string]float64, len(filesContainingWord))
			}

			tfIdf[fileId][word] = tf[fileId][word] * idf[word]
		}
	}

	return TfIdf{
		Tf:    tf,
		Idf:   idf,
		TfIdf: tfIdf,
	}
}

func tf(wordsCount int, files []path) map[fileId]map[word]float64 {
	tf := make(map[fileId]map[word]float64, wordsCount)

	texts := loadFileTexts(files)

	for fileId, text := range texts {
		concretteTf := make(map[word]float64)
		wordsCounter := make(map[word]int)
		for _, word := range text {
			if value, exists := wordsCounter[word]; !exists {
				wordsCounter[word] = 1
			} else {
				wordsCounter[word] = value + 1
			}
		}

		documentWordsCount := len(text)
		for word, wordCount := range wordsCounter {
			concretteTf[word] = float64(wordCount) / float64(documentWordsCount)
		}

		tf[fileId] = concretteTf
	}

	return tf
}

func idf(invertedIndex InvertedIndex, totalFilesCount float64) map[word]float64 {
	idf := make(map[word]float64, len(invertedIndex))

	for word, filesContainingWord := range invertedIndex {
		idf[word] = math.Log(float64(totalFilesCount) / float64(len(filesContainingWord)))
	}

	return idf
}

func getFilePaths() []path {
	paths := make([]path, 0)

	files, err := filepath.Glob(filepath.Join(DOCUMENTS_FOLDER, "*.txt"))
	if err != nil {
		log.Fatalln(err)
	}

	numberPattern := regexp.MustCompile(`^\d+\.txt$`)

	for _, file := range files {
		filename := filepath.Base(file)
		if numberPattern.MatchString(filename) {
			paths = append(paths, file)
		}
	}

	return paths
}

func getFileId(file path) fileId {
	base := filepath.Base(file)
	idStr := strings.TrimSuffix(base, ".txt")

	return idStr
}

func loadFileTexts(files []path) map[fileId][]word {
	docTexts := make(map[fileId][]word, len(files))

	for _, file := range files {
		fileId := getFileId(file)
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalln(err)
		}

		docTexts[fileId] = strings.Fields(string(content))
	}

	return docTexts
}

func loadInvertedIndex(invertedIndexPath path) (index InvertedIndex) {
	data, err := ioutil.ReadFile(invertedIndexPath)
	if err != nil {
		log.Fatalf("Error reading inverted_index.json: %v", err)
	}

	if err := json.Unmarshal(data, &index); err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}
	return
}

func saveResults(index TfIdf) {
	saveTf(index.Tf)
	saveIdf(index.Idf)
	saveTfIdf(index.TfIdf)
}

func saveTf(tf map[fileId]map[word]float64) {
	file, err := os.Create(filepath.Join(OUTPUT_FOLDER_PATH, "tf.csv"))
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	row := []string{"fileId", "word", "tf"}

	writer.Write(row)
	for fileId, concretteTf := range tf {
		row[0] = fileId

		sortedWords := make([]string, 0, len(concretteTf))
		for word := range concretteTf {
			sortedWords = append(sortedWords, word)
		}
		sort.Strings(sortedWords)

		for _, word := range sortedWords {
			row[1] = word
			row[2] = fmt.Sprintf("%.6f", concretteTf[word])

			writer.Write(row)
		}
	}
}

func saveIdf(idf map[word]float64) {
	file, err := os.Create(filepath.Join(OUTPUT_FOLDER_PATH, "idf.csv"))
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	sortedWords := make([]string, 0, len(idf))
	for word := range idf {
		sortedWords = append(sortedWords, word)
	}
	sort.Strings(sortedWords)

	row := []string{"word", "idf"}

	writer.Write(row)
	for _, word := range sortedWords {
		row[0] = word
		row[1] = fmt.Sprintf("%.6f", idf[word])

		writer.Write(row)
	}
}

func saveTfIdf(tfIdf map[fileId]map[word]float64) {
	file, err := os.Create(filepath.Join(OUTPUT_FOLDER_PATH, "tf-idf.csv"))
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	row := []string{"fileId", "word", "tf-idf"}

	writer.Write(row)
	for fileId, concretteTfIdf := range tfIdf {
		row[0] = fileId

		sortedWords := make([]string, 0, len(concretteTfIdf))
		for word := range concretteTfIdf {
			sortedWords = append(sortedWords, word)
		}
		sort.Strings(sortedWords)

		for _, word := range sortedWords {
			row[1] = word
			row[2] = fmt.Sprintf("%.6f", concretteTfIdf[word])

			writer.Write(row)
		}
	}
}
