package parser

import (
	"colly"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"unicode"

	"golang.org/x/net/html"
)

const targetWordsCount int = 1000

type htmlTextToFileWriter struct {
	wg               sync.WaitGroup
	toParseQueue     chan colly.HTMLElement
	parsedPages      int64
	gotRequestToStop uint32
}

type indexMeta struct {
	fileId int64
	url    string
}

func New(context context.Context, distanationPath string, workersCount int) (*htmlTextToFileWriter, error) {
	if workersCount < 1 {
		return nil, errors.New("pass at least 1 worker")
	}
	if distanationPath == "" {
		return nil, errors.New("provide dist dir")
	}

	w := htmlTextToFileWriter{
		wg:               sync.WaitGroup{},
		toParseQueue:     make(chan colly.HTMLElement, workersCount*2),
		parsedPages:      0,
		gotRequestToStop: 0,
	}

	cleanUpDir(distanationPath)
	setupWorkersAndFinish(context, &w, distanationPath, workersCount)

	return &w, nil
}

func cleanUpDir(distanationPath string) {
	exists := false
	_, err := os.Stat(distanationPath)
	if err == nil {
		exists = true
	} else if errors.Is(err, fs.ErrNotExist) {
		exists = false
	} else {
		log.Fatalln(err)
	}

	if exists {
		err := os.RemoveAll(distanationPath)
		if err != nil {
			log.Fatalln(err)
		}
	}

	err = os.Mkdir(distanationPath, os.ModeDir)
	if err != nil {
		log.Fatalln(err)
	}
}

func setupWorkersAndFinish(ctx context.Context, w *htmlTextToFileWriter, distanationPath string, workersCount int) {

	toIndexChan := make(chan indexMeta, workersCount)

	for range workersCount {
		w.wg.Add(1)

		go func() {
			defer w.wg.Done()
			sb := &strings.Builder{}

			for {
				select {
				case <-ctx.Done():
					return
				case html, ok := <-w.toParseQueue:
					if !ok {
						return
					}

					w.parseOne(sb, &distanationPath, &html, toIndexChan)
				}
			}
		}()
	}

	go writeIndex(ctx, distanationPath, toIndexChan)
}

func writeIndex(ctx context.Context, dir string, toWrite <-chan indexMeta) {
	indexPath := fmt.Sprintf("%s\\index.txt", dir)
	file, err := os.Create(indexPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	_, err = file.WriteString("id,url\n")
	if err != nil {
		log.Fatalln(err)
	}
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case info, ok := <-toWrite:
			if !ok {
				break LOOP
			}

			_, err = file.WriteString(fmt.Sprintf("%d,%s\n", info.fileId, info.url))
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func isNotRussianHtml(e *colly.HTMLElement) bool {
	return !e.DOM.Is("html[lang='ru']")
}

func (w *htmlTextToFileWriter) parseOne(sb *strings.Builder, dir *string, e *colly.HTMLElement, toIndexFile chan<- indexMeta) {
	if isNotRussianHtml(e) {
		return
	}

	text := parseElement(sb, e)

	if !hasEqualOrMoreThanNWords(&text, targetWordsCount) {
		return
	}

	fileNumber := atomic.AddInt64(&w.parsedPages, 1)
	fullFilePath := fmt.Sprintf("%s\\%d.txt", *dir, fileNumber)
	file, err := os.Create(fullFilePath)

	if err != nil {
		atomic.AddInt64(&w.parsedPages, -1)
		println(err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(text)

	if err != nil {
		println(err)

		atomic.AddInt64(&w.parsedPages, -1)
		err = os.Remove(fullFilePath)
		if err != nil {
			println(err)
		}
	}

	toIndexFile <- indexMeta{
		fileId: fileNumber,
		url:    e.Request.URL.String(),
	}
}

func hasEqualOrMoreThanNWords(text *string, n int) bool {
	count := 0
	inWord := false

	for _, rune := range *text {
		if unicode.IsSpace(rune) || unicode.IsPunct(rune) {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}

		if count >= n {
			return true
		}
	}

	return count >= n
}

func parseElement(sb *strings.Builder, e *colly.HTMLElement) string {
	sb.Reset()

	for _, node := range e.DOM.Nodes {
		visit(sb, node)
	}

	return sb.String()
}

func visit(sb *strings.Builder, n *html.Node) {
	if n == nil {
		return
	}

	switch n.Type {
	case html.ElementNode:
		skipElement := n.Data == "script" || n.Data == "style" || n.Data == "noscript" || n.Data == "meta" || n.Data == "link"
		if skipElement {
			return
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(sb, child)
		}

	case html.TextNode:
		for _, word := range strings.Fields(n.Data) {
			sb.WriteString(word)
			sb.WriteRune(' ')
		}
	}
}

func (w *htmlTextToFileWriter) Complete() (err error) {
	swapped := atomic.CompareAndSwapUint32(&w.gotRequestToStop, atomic.LoadUint32(&w.gotRequestToStop), 1)
	if swapped {
		close(w.toParseQueue)
	} else {
		err = errors.New("can not complete twice")
	}

	return
}

func (w *htmlTextToFileWriter) Wait() {
	w.wg.Wait()
}

func (w *htmlTextToFileWriter) Process(e colly.HTMLElement) (err error) {
	if atomic.LoadUint32(&w.gotRequestToStop) == 1 {
		return errors.New("parser is finishing its work")
	}

	defer func() {
		if panic := recover(); panic != nil {
			err = errors.New("parser is disposed")
		}
	}()

	w.toParseQueue <- e

	return
}
