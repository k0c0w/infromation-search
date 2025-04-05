package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type url = string
type fileId = string
type void = struct{}

type FileMeta struct {
	Id  fileId `json:"id"`
	Url url    `json:"url"`
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalln("Usage: go run main.go /path/to/index.json /path/to/output/dir")
	}
	indexPath := os.Args[1]
	outputDir := os.Args[2]

	dir := filepath.Dir(indexPath)

	invertedIndex := simplifyIndex(createIndexFromFiles(dir, parseMetadata(indexPath)))

	outputPath := filepath.Join(outputDir, "inverted_index.json")
	outputData, err := json.MarshalIndent(invertedIndex, "", "  ")
	if err != nil {
		log.Fatalln("Error marshaling inverted index:", err)
	}

	if err := os.WriteFile(outputPath, outputData, 0777); err != nil {
		log.Fatalln("Error writing inverted_index.json:", err)
	}

	txtOutPut(outputDir, invertedIndex)
}

func txtOutPut(dir string, index map[string][]fileId) {
	file, err := os.Create(path.Join(dir, "inverted_index.txt"))
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	keys := make([]string, 0, len(index))
	for key := range index {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		fileIds := strings.Join(index[key], ", ")
		writer.WriteString(fmt.Sprintf("%s: %s\n", key, fileIds))
		if i%3 == 0 {
			writer.Flush()
		}
	}

	writer.Flush()
}

func parseMetadata(indexPath string) (entries []*FileMeta) {
	data, err := ioutil.ReadFile(indexPath)
	if err != nil {
		log.Fatalln("Error reading index.json:", err)
	}

	if err := json.Unmarshal(data, &entries); err != nil {
		log.Fatalln("Error parsing index.json:", err)
	}

	return
}

func createIndexFromFiles(dir string, entries []*FileMeta) (invertedIndex map[string]map[fileId]void) {
	invertedIndex = make(map[string]map[fileId]void)

	for _, entry := range entries {
		txtPath := filepath.Join(dir, fmt.Sprintf("%s.txt", entry.Id))

		file, err := os.Open(txtPath)
		if err != nil {
			log.Printf("Error reading file %s: %v\n", txtPath, err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanWords)

		for scanner.Scan() {
			word := scanner.Text()
			wordIndex, wordRecordExists := invertedIndex[word]
			if !wordRecordExists {
				wordIndex = make(map[fileId]void)
				invertedIndex[word] = wordIndex
			}

			if _, idExists := wordIndex[entry.Id]; !idExists {
				wordIndex[entry.Id] = void{}
			}
		}
	}

	return
}

func simplifyIndex(invertedIndex map[string]map[fileId]void) (simplified map[string][]fileId) {
	simplified = map[string][]fileId{}

	for word, ids := range invertedIndex {
		idArr := make([]fileId, 0, len(ids))

		for id := range ids {
			idArr = append(idArr, id)
		}

		simplified[word] = idArr
	}

	return
}
