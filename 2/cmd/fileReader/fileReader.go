package fileReader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log"
	"os"
	"path/filepath"
)

type fileMetaData struct {
	Id string `json:"id"`
}

func readJsonFile(filePath string) ([]*fileMetaData, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Parse the JSON data
	var records []*fileMetaData
	err = json.Unmarshal(bytes, &records)
	if err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	return records, nil
}

type fileReader struct {
	path string
}

func (r *fileReader) GetFilePath() string {
	return r.path
}

func (r *fileReader) Words() (iter.Seq[string], error) {
	file, err := os.Open(r.path)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)

	return func(yield func(string) bool) {
		defer file.Close()

		for scanner.Scan() {
			if !yield(scanner.Text()) {
				break
			}
		}
	}, nil
}

type filesIterator struct {
	filePaths []string
}

func New(indexFilePath string) (*filesIterator, error) {
	lines, err := readJsonFile(indexFilePath)
	if err != nil {
		return nil, err
	}

	filePaths := make([]string, len(lines))

	dirPath := filepath.Dir(indexFilePath)
	for i, fileInfo := range lines {

		filePaths[i] = filepath.Join(dirPath, fmt.Sprintf("%s.txt", fileInfo.Id))
	}

	return &filesIterator{
		filePaths: filePaths,
	}, nil
}

func (r *filesIterator) WordReaders() iter.Seq[fileReader] {
	return func(yield func(fileReader) bool) {
		for _, path := range r.filePaths {
			if !yield(fileReader{path: path}) {
				break
			}
		}
	}
}
