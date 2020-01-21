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

func classic(uuid []string) ([]string, string){
    start := time.Now()
    reqChan := make(chan reqItem)
    stagingChan := make(chan reqItem) 
    responseChan := make(chan respItem)
    go func() {
        for i, u := range uuid {
            go func(i int, u string) {
                //uuids are sent along with their base64 encoding and their input order on the request channel
                reqChan <- reqItem{order: i, uuid: u, auth: b64.URLEncoding.EncodeToString([]byte(u))} 
            }(i, u)
        }
    }()
    //to initialize the process, 5 requests are staged
    for i:=0; i<5; i++{
        go func(){
            c := <- reqChan
            stagingChan <- c
        }()
    }
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
    res := uuid
    for{
        select {
        case r := <- responseChan:
            //put response body into the result slice at the same index as the uuid was entered as input
            res[r.order] = r.body
            fmt.Println("uuid:", r.uuid, "result:", r.body)
            resCount++
            if resCount == len(uuid){
                //when there is a result for all uuids return the result and the time elapsed since the beginning of the function
                tim := fmt.Sprintf("%v", time.Since(start))
                return res, tim
            }
        }
    }

}

func timingMethod(uuid []string)([]string, string){
    start := time.Now()
    reqChan := make(chan reqItem)
    responseChan := make(chan respItem)
    brk := 600*time.Millisecond //time between each batch of 4 requests
    inc := 100*time.Millisecond //time between each request within a batch
    count := 0 //to track the number of requests already in the rurrent batch
    go func(){
        for i, u := range uuid {
            //uuids are sent along with their base64 encoding and their input order on the request channel
            reqChan <- reqItem{order: i, uuid: u, auth: b64.URLEncoding.EncodeToString([]byte(u))}
        }
    }()
    //the below block continuously sends out sets of 4 requests according to the values in brk and inc
    go func(){
        for{
            select{
            case req := <-reqChan:
                //send 4 requests seperated by duration of inc 
                go call(req, responseChan, reqChan)
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
    res := uuid
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

func call(item reqItem, resChan chan respItem, reqChan chan reqItem){
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
        resChan <- respItem{item.order, item.uuid, string(body)}
    } else if resp.StatusCode == 429 {
        //if the request failed due to Too May Requests, retry later
        reqChan <- item
    } else {
        resChan <- respItem{item.order, item.uuid, "Request Failed"}
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

func main(){

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
        var result []string
        var tim string
        //start stopwatch 
        start = time.Now()
        if mode == "Classic"{
            //retrieve data using classic function according to user settings 
            fmt.Println("Requesting UUIDs using Classic method")
            result, tim = classic(uuid)
        } else if mode == "Timing"{
            //retrieve data using timingMethod function according to user settings 
            fmt.Println("Requesting UUIDs using Timing method")
            result, tim = timingMethod(uuid)
        }
        fmt.Println("_____________________________________")
        fmt.Println("Result:")
        fmt.Println(result)
        fmt.Println("_____________________________________")
        fmt.Println("Processing Time:")
        fmt.Println(tim)
        fmt.Println("_____________________________________")
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
