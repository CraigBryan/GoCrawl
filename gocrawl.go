package main

import "fmt"
import "github.com/franela/goreq"

type CrawlResult struct {
    Req *goreq.Request
    Item *ParsedT1Thing
    Persist interface{}
}

func (result CrawlResult) PrettyPrint() {
    fmt.Printf("Result: \n")
    if result.Req != nil {
        fmt.Printf("\tRequest: %s, %s\n", result.Req.Uri, result.Req.Method)
    }
    
    if result.Item != nil {
        fmt.Printf("\tItem: %+v\n", result.Item)
    }
}

var Downloaders [1]Downloader

func RunDownloaders(
    resultChan chan CrawlResult, respChan chan *goreq.Response,
) {
    var resp *goreq.Response
    var req *goreq.Request
    var ok bool
    for rslt := range resultChan {
        ok = true

        if rslt.Req != nil {
            for _, dler := range Downloaders {
                req, resp = dler.ProcessRequest(rslt.Req, nil)
                if req == nil && resp == nil {
                    // fmt.Println("Nothing back from downloader")
                    ok = false
                    break
                }
            }
            for _, dler := range Downloaders {
                resp = dler.ProcessResponse(resp)
                if resp == nil {
                    // panic("No response from downloader")
                    ok = false
                }
            }
            if ok {
                respChan <- resp
            }
        } else {
            fmt.Printf("Got Item: \n")
            rslt.Item.PrettyPrint()
        }
    }
}

func main() {
    // Number of downloading/item processing threads
    workers := 10
    
    // Figure out why this deadlocks with more reasonable buffer sizes
    resultChan := make(chan CrawlResult, 500)
    respChan := make(chan *goreq.Response, 500)
    
    Downloaders[0] = SimpleDownloader{}
    // Downloaders[1] = RateLimiter{}

    for i := 0; i < workers; i++ {
        go RunDownloaders(resultChan, respChan)
    }
    
    rs := NewRedditSpider(resultChan)
    rs.StartCrawl()
    
    for resp := range respChan {
        rs.Parse(resp)
    }
}