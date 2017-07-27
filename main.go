package main

import (
	"fmt"
	"sync"
	"net/http"
	"golang.org/x/net/html"
	"strings"
)

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

type Cache struct {
	visited map[string]bool
	mux sync.Mutex
}


// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher,
ch chan response, cache Cache) {
	defer close(ch)
	if depth <= 0 {
		return
	}
	cache.mux.Lock()
	if cache.visited[url] {
		cache.mux.Unlock()
		return
	}
	cache.visited[url] = true
	cache.mux.Unlock()

	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println(err)
		return
	}

	ch <- response{url, body}
	result := make([]chan response, len(urls))
	for i, u := range urls {
		result[i] = make(chan response)
		go Crawl(u, depth-1, fetcher, result[i], cache)
	}

	for i := range result {
		for resp := range result[i] {
			ch <- resp
		}
	}

	return
}

func main() {
	var ch = make(chan response)
	go Crawl("http://golang.org/", 4, fetcher, ch, Cache{visited: make(map[string] bool)})
	for resp := range ch {
		fmt.Printf("found: %s %q\n", resp.url, resp.body)
	}
}

type response struct {
	url string
	body string
}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult
type realFetcher map[string]*realResult

type fakeResult struct {
	body string
	urls []string
}

type realResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

func Fetch(url string) (string, []string, error) {
	fmt.Printf("Visiting %s.\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", nil, fmt.Errorf("not found: %s", url)
	}

	defer resp.Body.Close()
	z := html.NewTokenizer(resp.Body)

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return "", nil, fmt.Errorf("not found: %s", url)
		}

		if tt == html.StartTagToken {
			t := z.Token()

			if t.Data == "a" {
				for _, a := range t.Attr {
					if a.Key == "href" {

						// if link is within jeremywho.com
						if strings.HasPrefix(a.Val, "https://jeremywho.com") {
							fmt.Printf("found: %s \n", a.Val)
						}
					}
				}
			}
		}
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