package crawler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"colly"
)

type url = string

type HtmlBodyProccessor interface {
	Process(e colly.HTMLElement) error
	Complete() error
}

type WebCrawler struct {
	isCrawling        int32
	workersCount      int
	responseProcessor HtmlBodyProccessor
	visitedUrls       sync.Map
}

func New(htmlBodyProccessor HtmlBodyProccessor, workersCount int) (*WebCrawler, error) {
	if workersCount <= 0 {
		return nil, errors.New("wrong parameter value")
	} else if htmlBodyProccessor == nil {
		return nil, errors.New("provide proccessor")
	}

	crawler := WebCrawler{
		isCrawling:        0,
		workersCount:      workersCount,
		responseProcessor: htmlBodyProccessor,
		visitedUrls:       sync.Map{},
	}

	return &crawler, nil
}

func (crawler *WebCrawler) setStopFlag() {
	addr := &crawler.isCrawling
	stopped := atomic.CompareAndSwapInt32(addr, atomic.LoadInt32(addr), 0)
	for ; !stopped; stopped = atomic.CompareAndSwapInt32(addr, atomic.LoadInt32(addr), 0) {
	}
}

func (crawler *WebCrawler) Crawl(ctx context.Context, entryUrls []url) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	addr := &crawler.isCrawling
	crawlStarted := atomic.CompareAndSwapInt32(addr, atomic.LoadInt32(addr), 1)
	if !crawlStarted {
		return errors.New("crawler has been aldready stared")
	}
	defer crawler.setStopFlag()

	c := colly.NewCollector(
		colly.StdlibContext(ctx),
		colly.MaxDepth(1),
		colly.UserAgent("Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36"),
		colly.Async(true),
	)
	c.SetRequestTimeout(30 * time.Second)

	c.Limit(&colly.LimitRule{
		Parallelism: crawler.workersCount,
		RandomDelay: time.Duration(time.Duration.Seconds(1)),
		Delay:       time.Duration(time.Duration.Seconds(1)),
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if isLocalLink := strings.HasPrefix(href, "/"); isLocalLink {
			href = fmt.Sprintf("%s://%s%s", e.Request.URL.Scheme, e.Request.Host, href)
		}

		if isHttpRef := strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://"); isHttpRef {
			if _, visited := crawler.visitedUrls.LoadOrStore(href, struct{}{}); !visited {
				c.Visit(href)
			}
		}
	})

	c.OnHTML("html", func(h *colly.HTMLElement) {
		err := crawler.responseProcessor.Process(*h)
		if err != nil {
			log.Println(err)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println(err)
	})

	for _, url := range entryUrls {
		c.Visit(url)
	}

	c.Wait()
	crawler.responseProcessor.Complete()
	log.Println("Crawling completed.")

	return nil
}
