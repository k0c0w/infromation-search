package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const IDF_CSV_PATH path = "C:\\CustomDesktop\\informations search\\4\\output\\idf.csv"
const TF_IDF_CSV_PATH path = "C:\\CustomDesktop\\informations search\\4\\output\\tf-idf.csv"
const FILE_TO_URL_JSON_PATH path = "C:\\CustomDesktop\\informations search\\2\\output\\index.json"
const MAX_OUTPUT int = 10

type fileId = string
type url = string
type word = string
type path = string

type DocIndex struct {
	Id  fileId `json:"id"`
	Url url    `json:"url"`
}

type SearchResult struct {
	FileId fileId
	Score  float64
	Url    url
}

func (sr *SearchResult) String() string {
	return fmt.Sprintf("Документ: %s\tСходство: %.6f\tUrl: %s", sr.FileId, sr.Score, sr.Url)
}

func main() {
	idf, tfIdf, indexMap := load()
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	var input string
	for {
		fmt.Print("Введите запрос: ")
		input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
		words := strings.Fields(strings.ToLower(input))
		queryVector := make(map[word]float64)

		for _, word := range words {
			queryVector[word]++
		}
		for word := range queryVector {
			if idfValue, exists := idf[word]; exists {
				queryVector[word] *= idfValue
			} else {
				delete(queryVector, word)
			}
		}

		var results []SearchResult
		for fileId, tfIdfVector := range tfIdf {
			similarityScore := cosineSimilarity(queryVector, tfIdfVector)
			if similarityScore > 0 {
				results = append(results, SearchResult{
					FileId: fileId,
					Score:  similarityScore,
					Url:    indexMap[fileId],
				})
			}
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})

		var cutoff int
		if len := len(results); len < MAX_OUTPUT {
			cutoff = len
		} else {
			cutoff = MAX_OUTPUT
		}
		results = results[:cutoff]

		writer.WriteRune('\n')
		for _, result := range results {
			writer.WriteString(result.String())
			writer.WriteRune('\n')
		}
		writer.WriteRune('\n')
		writer.Flush()
	}
}

func cosineSimilarity(vec1, vec2 map[word]float64) float64 {
	var productSum, normA, normB float64

	for word, a := range vec1 {
		b := vec2[word]
		productSum += a * b
	}

	for _, a := range vec1 {
		normA += a * a
	}

	for _, b := range vec2 {
		normB += b * b
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return productSum / (math.Sqrt(normA) * math.Sqrt(normB))
}

func loadFileToUrlMapping() map[fileId]url {
	file, err := os.ReadFile(FILE_TO_URL_JSON_PATH)
	if err != nil {
		log.Fatalln(err)
	}

	var index []DocIndex
	err = json.Unmarshal(file, &index)
	if err != nil {
		log.Fatalln(err)
	}

	result := make(map[string]string)
	for _, item := range index {
		result[item.Id] = item.Url
	}
	return result
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func loadTfIdf() map[fileId]map[word]float64 {
	file, err := os.Open(TF_IDF_CSV_PATH)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()

	tfIdf := make(map[fileId]map[word]float64)
	for _, record := range records[1:] {
		fileId := record[0]
		word := record[1]
		value := parseFloat(record[2])

		concretteTfIdf, exists := tfIdf[fileId]
		if !exists {
			concretteTfIdf = make(map[string]float64)
			tfIdf[fileId] = concretteTfIdf
		}
		concretteTfIdf[word] = value
	}

	return tfIdf
}

func loadIdf() map[word]float64 {
	file, err := os.Open(IDF_CSV_PATH)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()

	idf := make(map[word]float64)
	for _, record := range records[1:] {
		word := record[0]
		val := parseFloat(record[1])
		idf[word] = val
	}

	return idf
}

func load() (idf map[word]float64, tfIdf map[fileId]map[word]float64, indexMap map[fileId]url) {
	idf = loadIdf()
	tfIdf = loadTfIdf()
	indexMap = loadFileToUrlMapping()

	return
}
