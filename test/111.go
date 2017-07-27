package main

import (
	"fmt"
	"sync"
)

type SafeMap struct {
	urls map[string]bool
	mux  sync.Mutex
}

type link struct {
	url  string
	body string
}

//var urlMap = SafeMap{urls : make(map[string]bool)}

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher, ch chan link, safeMap SafeMap) {
	defer close(ch)

	if depth <= 0 {
		return
	}

	safeMap.mux.Lock()
	if safeMap.urls[url] {
		safeMap.mux.Unlock()
		return
	}

	safeMap.urls[url] = true
	safeMap.mux.Unlock()

	body, urls, err := fetcher.Fetch(url)

	if err != nil {
		fmt.Println(err)
		return
	}

	//linkNew := link{url: url, body: body}
	ch <- link{url, body}

	chanels := make([]chan link, len(urls))

	for i, u := range urls {
		chanels[i] = make(chan link)
		go Crawl(u, depth - 1, fetcher, chanels[i], safeMap)
	}

	for i := range chanels {
		for resp := range chanels[i] {
			ch <- resp
		}
	}

	return
}

func main() {
	var ch = make(chan link)
	Crawl("http://golang.org/", 4, fetcher, ch, SafeMap{urls : make(map[string]bool)})

	for link := range ch {
		fmt.Printf("found: %s %q\n", link.url, link.body)
	}

}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"http://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"http://golang.org/pkg/",
			"http://golang.org/cmd/",
		},
	},
	"http://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"http://golang.org/",
			"http://golang.org/cmd/",
			"http://golang.org/pkg/fmt/",
			"http://golang.org/pkg/os/",
		},
	},
	"http://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"http://golang.org/",
			"http://golang.org/pkg/",
		},
	},
	"http://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"http://golang.org/",
			"http://golang.org/pkg/",
		},
	},
}
