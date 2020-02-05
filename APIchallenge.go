package main

import (
	"fmt"
	"net/http"
	b64 "encoding/base64"
    "log"
    "time"
    "io/ioutil"
    "os"
    "strings"
    "os/exec"
    "runtime"
    "encoding/json"
    "path/filepath"
    "github.com/kirsle/configdir"
)

type reqItem struct {
    order int
    uuid string
    auth string
}

type respItem struct{
    order int
    uuid string
    body string
}

type cacheQuery struct{
    order int
    uuid string
}

type cacheStore struct {
    Cache map[string]string `json:"cache"`
}

var cacheQueryChan = make(chan cacheQuery)
var cacheEntryChan = make(chan respItem)
var cacheSaveChan = make(chan bool)
var reqChan = make(chan reqItem)
var responseChan = make(chan respItem)

func classic(uuid []string) ([]string, string){
    start := time.Now()
    stagingChan := make(chan reqItem)
    go func() {
        for i, u := range uuid {
            go func(i int, u string) {
                //uuids are sent along with their input order to the cache
                cacheQueryChan <- cacheQuery{order: i, uuid: u} // cache will forward request to the reqChan or will send response to responseChan
            }(i, u)
        }
    }()
    //to initialize the process, 5 requests are staged
    i := 0
    go func(){
        for i<5{
            select{
                case c := <- reqChan:
                stagingChan <- c
                i++
            }
        }
    }()
    go func(){
        for{
            select {
            case this := <- stagingChan:
                go func() {
                    //create the request
                    req, err := http.NewRequest("GET", "https://challenges.qluv.io/items/"+ this.uuid, nil)
                    if err != nil {
                        fmt.Println(err)
                        return
                    }
                    //set the header
                    req.Header.Set("Authorization", this.auth)
                    resp, err := http.DefaultClient.Do(req)
                    if err != nil {
                        fmt.Println(err)
                        return
                    }
                    //get the response body
                    var strBody string
                    if resp.StatusCode == 200  {
                        body, err := ioutil.ReadAll(resp.Body)
                        if err != nil {
                            fmt.Println(err)
                            return
                        }
                        strBody = string(body)
                    } else {
                        strBody = "Request Failed"
                        
                    }
                    //send the reponse info to the response channel
                    result := respItem{this.order, this.uuid, strBody}
                    responseChan <- result
                    // cache it
                    cacheEntryChan <- result
                    //stage another request since the API is now availible to service another request
                    next := <- reqChan
                    stagingChan <- next
                    defer resp.Body.Close()
                }()
            }
        }
    }()
    
    resCount := 0
    //make copy of uuid slice to ensure that the result slice is of the same size
    res := make([]string, len(uuid))
    for{
        select {
        case r := <- responseChan:
            //put response body into the result slice at the same index as the uuid was entered as input
            res[r.order] = r.body
            fmt.Println("uuid:", r.uuid, "result:", r.body)
            resCount++
            if resCount == len(uuid){
                tim := fmt.Sprintf("%v", time.Since(start))
                return res, tim
            }
        }
    }

}

func timingMethod(uuid []string)([]string, string){
    start := time.Now()
    brk := 600*time.Millisecond //time between each batch of 4 requests
    inc := 100*time.Millisecond //time between each request within a batch
    count := 0 //to track the number of requests already in the rurrent batch
    go func(){
        for i, u := range uuid {
            //uuids are sent along with their input order to the cache
            cacheQueryChan <- cacheQuery{order: i, uuid: u}
        }
    }()
    //the below block continuously sends out sets of 4 requests according to the values in brk and inc
    go func(){
        for{
            select{
            case req := <-reqChan:
                //send 4 requests seperated by duration of inc 
                go call(req)
                if count < 4{
                    time.Sleep(inc)
                    count++
                } else{
                    //then wait a duration of brk before sending the next batch of 4
                    time.Sleep(brk)
                    count = 0
                }
            }
        }
    }()
    resCount := 0
    res := make([]string, len(uuid))
    for{
        select {
        case r := <- responseChan:
            //put response body into the result slice at the same index as the uuid was entered as input
            resCount++
            fmt.Println("uuid:", r.uuid, "result:", r.body)
            res[r.order] = r.body
            if resCount == len(uuid){
                //when there is a result for all uuids return the result and the time elapsed since the beginning of the function
                tim := fmt.Sprintf("%v", time.Since(start))
                return res, tim
            }
        }
    }
}

func call(item reqItem){
    //create the request
    req, err := http.NewRequest("GET", "https://challenges.qluv.io/items/"+item.uuid, nil)
    if err != nil {
        fmt.Println(err)
        return
    }
    //set the header
    req.Header.Set("Authorization", item.auth)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        fmt.Println(err)
        return
    }
    var body string
    //check status code
    if resp.StatusCode == 200  {
        bodyByte, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            fmt.Println(err)
            return
        }  
        body = string(bodyByte)
        result := respItem{item.order, item.uuid, string(body)}
        responseChan <- result
        //store in cache
        cacheEntryChan <- result
    } else if resp.StatusCode == 429 {
        //if the request failed due to Too May Requests, retry later
        reqChan <- item
    } else {
        result := respItem{item.order, item.uuid, "Request Failed"}
        responseChan <- result
        //store in cache
        cacheEntryChan <- result
    }
    defer resp.Body.Close()
}

//The openbrowser function is from github. Find it here: https://gist.github.com/hyg/9c4afcd91fe24316cbf0
func openbrowser(url string) {
    var err error

    switch runtime.GOOS {
    case "linux":
        err = exec.Command("xdg-open", url).Start()
    case "windows":
        err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    case "darwin":
        err = exec.Command("open", url).Start()
    default:
        err = fmt.Errorf("unsupported platform")
    }
    if err != nil {
        log.Fatal(err)
    }

}

func operateCache() {
    //one the cache begins opertation it checks the user's computer for saved cache data from a previous session
    path := configdir.LocalCache() // locates the user's local cache foilder
    err := configdir.MakePath(path) // Ensures the folder exists.
    if err != nil {
        path = ""
    }
    cacheFile := filepath.Join(path, "APIchallenge_cache.json")
    var cacheJson cacheStore
    var cache map[string]string
    // check if user's computer has data stored from previous session
    if _, err = os.Stat(cacheFile); !os.IsNotExist(err) {
        f, err := os.Open(cacheFile)
        if err != nil {
            fmt.Println("failed to read saved cache data")
            cache = make(map[string]string)
        } else {
            defer f.Close()
            //retrieve the stored cache data and initialize the cache with that data
            decoder := json.NewDecoder(f)
            decoder.Decode(&cacheJson)
            cache = cacheJson.Cache
        }
    } else {
        // if the user does not have saved data in cache.json initialize an empty cache
        cache = make(map[string]string)
    }
    for {
        select {
        case req := <- cacheQueryChan:
            go func() {
                if data, in := cache[req.uuid]; in {
                    //if the cache contains the requested uuid forward the response
                    responseChan <- respItem{order: req.order, uuid: req.uuid, body: data}
                } else {
                    //otherwise put it on the request channel to request it from the API
                    reqChan <- reqItem{order: req.order, uuid: req.uuid, auth: b64.URLEncoding.EncodeToString([]byte(req.uuid))}
                }
            }()
        case res := <- cacheEntryChan:
            if _ , in := cache[res.uuid]; !in {
                // store the retrieved result in the cache
                cache[res.uuid] = res.body
            }
        case <- cacheSaveChan:
            if !(path == ""){ 
                //if the program was able to find the user's local cache foilder, marshal the cache to json and store it there
                cacheJson = cacheStore{cache}
                f, err := os.Create(cacheFile)
                if err != nil {
                    fmt.Println("failed to save cache data")
                    break
                }
                defer f.Close()
                encoder := json.NewEncoder(f)
                encoder.Encode(&cacheJson)
                fmt.Println("Stored session cache data to ", cacheFile)
                fmt.Println("_____________________________________")
            }
        }
    }
}

func main(){
    go operateCache()
    status := ""
    var start time.Time
    openbrowser("http://localhost:8081/static/")
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, r.URL.Path[1:])
    })

    http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
        if status == "Loading"{
            //deny requests if another set of uuids is still processing
            fmt.Println("Request for data denied. Already processing a request. Please wait.")
            return
        }
        fmt.Println("Recieved submission")
        fmt.Println("_____________________________________")
        err := r.ParseForm()
        if err != nil {
            fmt.Fprintf(os.Stdout, "could not parse query: %v", err)
            w.WriteHeader(http.StatusBadRequest)
        }
        //get user settings
        mode := r.FormValue("mode")
        file := r.FormValue("file")
        //set status 
        status = "Loading"
        var uuid []string
        if file == ""{
            //recieve the uuid data from the browser's http request
            uuidStr := r.Header.Get("uuid")
            uuid = strings.Split(uuidStr, ",")
        } else {
            //recieve uuid data from the file on the user's computer
            byteContent, err := ioutil.ReadFile(file)
            if err != nil {
                log.Fatal(err)
            }
            uuidStr := string(byteContent)
            //parse file data into slice
            uuid = strings.Split(uuidStr, "\n")
                     
        }
        fmt.Println("UUIDs:")
        fmt.Println(uuid)
        fmt.Println("_____________________________________")
        var resultSet []string
        var tim string
        result := make([]string, len(uuid))

        //remove duplicates and store the indexes and order of each entry
        uuidCompressed := make(map[string][]int)
        uuidSet := make([]string, 0)
        for i, v := range uuid{
            indx, in := uuidCompressed[v]
            if !in {
                uuidSet = append(uuidSet, v)
                indx = make([]int,0)
            }
            uuidCompressed[v] = append(indx, i)
        }
        //start stopwatch 
        start = time.Now()
        if mode == "Classic"{
            //retrieve data using classic function according to user settings 
            fmt.Println("Requesting UUIDs using Classic method")
            resultSet, tim = classic(uuidSet)
        } else if mode == "Timing"{
            //retrieve data using timingMethod function according to user settings 
            fmt.Println("Requesting UUIDs using Timing method")
            resultSet, tim = timingMethod(uuidSet)
        }
        //save the updated cache to cache.json for future sessions
        cacheSaveChan <- true
        //build full results using resultSet and uuidCompressed
        for i, res := range resultSet{
            for _ , indx := range uuidCompressed[uuidSet[i]]{
                result[indx] = res
            }
        }
        fmt.Println("_____________________________________")
        fmt.Println("Result:")
        fmt.Println(result)
        fmt.Println("_____________________________________")
        fmt.Println("Processing Time:")
        fmt.Println(tim)
        //put data and final processing time into output.txt file
        result = append(result, tim)
        resultStr := strings.Join(result, ",")
        f, err := os.Create("static/output.txt")
        if err != nil {
            fmt.Println(err)
            return
        }
        _, err = f.WriteString(resultStr)
        if err != nil {
            fmt.Println(err)
            f.Close()
            return
        }
        err = f.Close()
        if err != nil {
            fmt.Println(err)
            return
        }
        //set status to done so browser will recieve data on next update
        status = "Done"
    })

    http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
        //check time elapsed
        tim := fmt.Sprintf("%v", time.Since(start))
        //check if valid request
        err := r.ParseForm()
        if err != nil {
            fmt.Fprintf(os.Stdout, "could not parse query: %v", err)
            w.WriteHeader(http.StatusBadRequest)
        }
        if status == "Loading"{
            //tell browser that the data is still loading and give a time elapsed update
            w.Header().Set("Location", "/static?result="+status+"&time="+tim)
        } else if status=="Done"{
            //tell broweser that the data is availible in output.txt.
            w.Header().Set("Location", "/static?result="+status)
        }

        w.WriteHeader(http.StatusFound)
    })
    fs := http.FileServer(http.Dir("static/"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))
    log.Fatal(http.ListenAndServe(":8081", nil))
}
