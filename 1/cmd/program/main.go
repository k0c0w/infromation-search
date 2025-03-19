package main

import (
	"context"
	"crawler"
	"errors"
	"fmt"
	"log"
	"os"
	"parser"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type cmdArgs struct {
	out     string
	urls    []string
	timeOut int
}

func (c cmdArgs) String() string {
	return fmt.Sprintf("Timeout %ds\n Out dir %s\n Entry urls %s", c.timeOut, c.out, strings.Join(c.urls, "\n\t"))
}

func indexOf(arr []string, pattern string) (index int) {
	index = -1

	for i, val := range arr {
		if val == pattern {
			index = i
			break
		}
	}

	return
}

func parseUrls(arg string) ([]string, error) {
	var err error
	urls := strings.Split(arg, ";")

	if len(urls) == 0 {
		err = errors.New("provide entry urls with separated by ;")
	}

	return urls, err
}

func parseArgs() (*cmdArgs, error) {
	args := os.Args[1:]

	urlsTagIndex := indexOf(args, "-urls")
	if urlsTagIndex == -1 {
		return nil, errors.New("provide entry urls with: -urls option[;option;...]")
	}

	urlsIndex := urlsTagIndex + 1
	if urlsIndex >= len(args) {
		_, err := parseUrls("")
		return nil, err
	}

	urls, err := parseUrls(args[urlsIndex])
	if err != nil {
		return nil, err
	}

	outTagIndex := indexOf(args, "-out")
	outIndex := outTagIndex + 1

	if outTagIndex == -1 || outIndex >= len(args) {
		return nil, errors.New("provide output dir path with: -out <..>")
	}

	tTagIndex := indexOf(args, "-t")
	tIndex := tTagIndex + 1

	if tIndex == -1 || tIndex >= len(args) {
		return nil, errors.New("provide timeout seconds with: -t <..>")
	}

	timeoutSeconds, err := strconv.Atoi(args[tIndex])
	if err != nil {
		return nil, err
	}

	return &cmdArgs{
		out:     args[outIndex],
		urls:    urls,
		timeOut: timeoutSeconds,
	}, nil
}

func main() {
	args, err := parseArgs()
	if err != nil {
		log.Fatalln(err)
	}
	println(args.String())

	halfWorkers := runtime.NumCPU() / 2
	if halfWorkers == 0 {
		halfWorkers = 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(args.timeOut)*time.Second)
	defer cancel()

	parser, err := parser.New(ctx, args.out, halfWorkers)
	if err != nil {
		println(err)
		return
	}

	crawler, err := crawler.New(parser, halfWorkers)
	if err == nil {
		crawler.Crawl(ctx, args.urls)
		parser.Wait()
	} else {
		println(err)
	}
}
