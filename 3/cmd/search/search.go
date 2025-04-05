package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
)

type url = string
type InvertedIndex map[string][]url

type Operation = int

const (
	OR  Operation = 1
	AND Operation = 2
	NOT Operation = 3
)

func main() {
	index := loadIndex("C:\\CustomDesktop\\informations search\\3\\output\\inverted_index.json")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	searchEngine(signalChan, index)
}

func searchEngine(interruptSignal <-chan os.Signal, index InvertedIndex) {
	google := New(index)
	in := bufio.NewReader(os.Stdin)

	for {
		select {
		case <-interruptSignal:
			return
		default:
			fmt.Print("ПОИСК: ")
			line, err := in.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				continue
			}
			trimmedQuery := strings.TrimSpace(line)
			result, err := google.Search(trimmedQuery)
			intResult := make([]int, 0, len(result))
			for _, result := range result {
				val, _ := strconv.Atoi(result)
				intResult = append(intResult, val)
			}
			sort.Ints(intResult)

			if err != nil {
				fmt.Println(err)
			} else {
				if len(result) == 0 {
					fmt.Println("[]")
				} else {
					fmt.Println(intResult)
				}
			}
		}
	}
}

func loadIndex(filename string) (index InvertedIndex) {
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}

	if err := json.Unmarshal(file, &index); err != nil {
		log.Fatalln(err)
	}

	return
}
