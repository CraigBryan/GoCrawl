package main

import "github.com/franela/goreq"
import "encoding/json"
import "fmt"
import "strings"

type Spider interface {
    Parse(*goreq.Response)
    EmitResult(CrawlResult)
    StartCrawl()
}

type RedditSpider struct {
    ConfigFile string
    ResultChan chan CrawlResult
    UserAgent string
}

func (rs RedditSpider) EmitResult(res CrawlResult) {
    rs.ResultChan <- res
}

func (rs RedditSpider) StartCrawl() {
    // TODO move UA and starting url into configs
    req := goreq.Request{
        Uri: "https://www.reddit.com/.json",
        UserAgent: rs.UserAgent,
    }

    rs.EmitResult(CrawlResult{Req: &req})
}

func NewRedditSpider(rChan chan CrawlResult) RedditSpider {
    rs := RedditSpider{
        ConfigFile: "reddit.conf",
        ResultChan: rChan,
        UserAgent: "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 " +
                   "(KHTML, like Gecko) Chrome/49.0.2623.112 Safari/537.36",
    }
    
    rs.Authorize()
    return rs
}

func (rs RedditSpider) Authorize() {
    // Todo OAuth2!
}

func (rs RedditSpider) Parse(resp *goreq.Response) {
    var rawJson json.RawMessage
    if err := resp.Body.FromJsonTo(&rawJson); err != nil {
        panic(fmt.Sprintf("Problem parsing response json from %s", resp.Uri))
    }
    
    var listThings []ParsedThing

    if err := json.Unmarshal(rawJson, &listThings); err != nil {
       var bodyThing ParsedThing
       if err := json.Unmarshal(rawJson, &bodyThing); err != nil {
           panic(err)
       }
       rs.ParseThing(bodyThing)
    } else {
        for _, thing := range listThings {
            rs.ParseThing(thing)   
        }
    }
}

func (rs RedditSpider) ParseThing(thing ParsedThing) {
    switch thing.Kind {
    case "Listing":
        rs.ParseListing(thing)
    case "t3":
        rs.ParseT3(thing)
    case "t1":
        rs.ParseT1(thing)
    case "more":
        fmt.Println("Ignoring 'more' thing kind")
    default:
        fmt.Printf("Unknown thing type %s\n", thing.Kind)
    }
}

func (rs RedditSpider) ParseListing(li ParsedThing) {
    var subThings []map[string]*json.RawMessage

    if err := json.Unmarshal(*li.Data["children"], &subThings); err != nil {
        panic(err)
    }
    
    var temp ParsedThing
    for _, rawThing := range subThings {
        temp = ParsedThing{}
        safeUnmarshalString(rawThing, "kind", &temp.Kind)
        safeUnmarshalJSON(rawThing, "data", &temp.Data)
        rs.ParseThing(temp)
    }
    
    // TODO pagination
}

func (rs RedditSpider) ParseT3(thing ParsedThing) {
    t3 := ParsedT3Thing{}
    safeUnmarshalInt(thing.Data, "score", &t3.Score)
    safeUnmarshalString(thing.Data, "permalink", &t3.Link)
    safeUnmarshalString(thing.Data, "author", &t3.Author)
    safeUnmarshalInt(thing.Data, "num_comments", &t3.NumComments)
    if (!strings.HasPrefix(t3.Link, "https://www.reddit.com") &&
        !strings.HasPrefix(t3.Link, "http://www.reddit.com")) {
        t3.Link = fmt.Sprintf("https://www.reddit.com%s", t3.Link)
    }
    
    if !strings.HasSuffix(t3.Link, ".json") {
        t3.Link = fmt.Sprintf("%s.json", t3.Link[:len(t3.Link) - 1])
    }
    rs.EmitResult(CrawlResult{
        Req: &goreq.Request{
            Uri: t3.Link,
            // TODO move UA to configs
            UserAgent: rs.UserAgent,
        },
    })
}

func (rs RedditSpider) ParseT1(thing ParsedThing) {
    t1 := ParsedT1Thing{}
    safeUnmarshalInt(thing.Data, "score", &t1.Score)
    safeUnmarshalString(thing.Data, "author", &t1.Author)
    
    res := CrawlResult{
        Req: nil,
        Item: &t1,
    }
    rs.EmitResult(res)
    
    var child map[string]*json.RawMessage
    if err := json.Unmarshal(*thing.Data["replies"], &child); err != nil {
        // No children
        return
    }
    
    subThing := ParsedThing{}
    safeUnmarshalString(child, "kind", &subThing.Kind)
    safeUnmarshalJSON(child, "data", &subThing.Data)
    rs.ParseThing(subThing)
}

type ParsedThing struct {
    Kind string
    Data map[string]*json.RawMessage
}

// t3's are top-level posts
type ParsedT3Thing struct {
    Score int
    Link string
    Author string
    NumComments int
}

func (t3 ParsedT3Thing) PrettyPrint() {
    fmt.Println("T3 thing")
    fmt.Printf("\tAuthor: %s\n", t3.Author)
    fmt.Printf("\tUpboats: %d\n", t3.Score)
    fmt.Printf("\tNumber of Comments: %d\n", t3.NumComments)
    fmt.Printf("\tLink: %s\n", t3.Link)
}

// t1's are comments
type ParsedT1Thing struct {
    Score int
    Author string
}

func (t1 ParsedT1Thing) PrettyPrint() {
    fmt.Println("\tT3 thing")
    fmt.Printf("\t\tAuthor: %s\n", t1.Author)
    fmt.Printf("\t\tUpboats: %d\n", t1.Score)
}

func safeUnmarshalString(
    raw map[string]*json.RawMessage,
    key string,
    out *string,
) {
    checkJSONKey(raw, key)
    if err := json.Unmarshal(*raw[key], &out); err != nil {
        panic(err)
    }
}

func safeUnmarshalInt(
    raw map[string]*json.RawMessage,
    key string,
    out *int,
) {
    checkJSONKey(raw, key)
    if err := json.Unmarshal(*raw[key], &out); err != nil {
        panic(err)
    }
}

func safeUnmarshalJSON(
    raw map[string]*json.RawMessage,
    key string,
    out *map[string]*json.RawMessage,
) {
    checkJSONKey(raw, key)
    if err := json.Unmarshal(*raw[key], out); err != nil {
        panic(err)
    }
}

func checkJSONKey(raw map[string]*json.RawMessage, key string) {
    _, ok := raw[key]
    if !ok {
        panic(fmt.Sprintf("Key \"%s\" not in JSON", key))
    }
}
