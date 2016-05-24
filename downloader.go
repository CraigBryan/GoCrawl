package main

import (
    "github.com/franela/goreq";
    "fmt"
)

type Downloader interface {
    ProcessRequest(
        *goreq.Request, *goreq.Response,
    ) (*goreq.Request, *goreq.Response)
    ProcessResponse(*goreq.Response) *goreq.Response
}

// SimpleDownloader simply resolves an http request
type SimpleDownloader struct {}

func (dl SimpleDownloader) ProcessRequest(
    req *goreq.Request, resp *goreq.Response,
) (*goreq.Request, *goreq.Response) {
    if resp != nil {
        return req, resp
    }
    if req == nil {
        return nil, nil
    }
    
    outResp, err := req.Do()
    if err != nil {
        if serr, ok := err.(*goreq.Error); ok {
            if serr.Timeout() {
                fmt.Printf("Request to %s timed out\n", req.Uri)
            }
        } else {
            fmt.Printf("Failed request to %s\n", req.Uri)
            panic(err)
        }
        return req, nil
    }
    
    fmt.Printf(
        "\nMade Request to %s Response status: %v\n\n",
        req.Uri,
        outResp.Status,
    )
    return req, outResp
}

func (dl SimpleDownloader) ProcessResponse(
    resp *goreq.Response,
) *goreq.Response {
    return resp
}

// // TODO implement
// // This ensures we don't crawl too fast
// // Currently reddit-specific
// type RateLimiter struct {}

// func (dl RateLimiter) ProcessRequest(
//     req *goreq.Request, resp *goreq.Response,
// ) (*goreq.Request, *goreq.Response) {
//     // TODO
//     return req, resp
// }

// func (dl RateLimiter) ProcessResponse(resp *goreq.Response) *goreq.Response {
//     // Todo update rate data
//     return resp
// }