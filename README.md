# API challenge
**Instructions:**
1. Clone or download this repository
2. Make sure to have Go downloaded. (https://golang.org/)
3. Open terminal and navigate to the directory where you stored this repository
4. To install a dependency, in terminal run: ```go get "github.com/kirsle/configdir"```
5. To run the program, in terminal run:

```
Reccomended:   go build APIchallenge.go
               ./APIchallenge
                   -or-
Aleternative:  go run APIchallenge.go
```
        
6. A browser window should open to http://localhost:8081/static/
7. The interface allows you to input UUIDs via text entry or through a .txt  file. UUIDs entered to the text box may be separated by newlines, commas or spaces. If they are entered as a .txt file, each UUID must be on its own line. No other separators are supported for file entries.
8. The interface also allows you to select one of two different methods for retrieving your data: 
    - Classic: Requests data for 5 UUIDs at a time. As soon as a response from one of those 5 is received, the next UUID is requested ensuring that there are 5 requests pending at any given time. This method ensures that the API’s request limit is never triggered.
    - Timing: Makes staggered requests according to a timing strategy that is based on estimates of the time it takes for a request to reach the API and the time it takes for the API to process a request. In service of higher throughput, this method doesn’t wait for responses before sending out the next request. This method triggers the API’s request limit at times. This method is slightly faster when requesting large batches of UUIDs. 
9. Click the “Get Result” button to fetch your desired data
10. Wait for your results to load. When they are finished loading they will appear on the right panel of the interface. Your latest result data will also be stored in the file static/output.txt if you prefer your result in a file (items in static/output.txt are comma-separated with the time it took to retrieve the data as the last item).
11. To input and request a new batch of UUIDs click the “Reset” button



**Notes:**
- I developed this project on my Mac but for testing purposes I cloned this repository and tried running it on my old laptop which has Windows. I found that every time I ran the command `go run APIchallenge.go` on Windows, I would have to authorize the program to use my network. However if I ran `go build APIchallenge.go` and then `./APIchallenge`, the first execution would require autorization, but subsequent calls to `./APIchallenge` would not require authorization. For this reason, I reccomend Window users use the second method for running the program.
- Only request 1 set of UUIDs at a time. Until the utility finishes processing your request, do not reload the window or make another request in another window. The utility can only process one set of UUIDs at a time. Your second request will be denied. The UI will display the results of the first request once they are ready.
- While data is being retrieved, the UI requests periodic updates from the APIchallenge.go program to see if the results are ready. The total elapsed time since the initial request is updated next to the UI’s “Stopwatch” element after each update. For additional information about the progress of your request view the terminal window.
- Be careful not to add additional characters to your input. 
- When  inputting data through Text Entry, additional trailing or preceding separators are fine because these will be removed but additional characters that are not the separator you selected may cause errors.
- When inputting data through File Entry, additional newlines such as a newline at the end of the file will be treated as UUID items by the program. Additional characters such as these as well as invalid UUIDs will give the result “Request Failed.”
- Invalid files and some other errors will cause the APIchallenge.go program to stop execution. If this happens, start over from step 4 above.
- Navigate to http://localhost:8081/ while APIchallenge.go is running to view the contents of all files in the repository from your browser
- See the [Info](https://github.com/eli-stewart/APIchallenge/tree/master/Info) folder in this repository for more information about the project. The file [reflections.pdf](https://github.com/eli-stewart/APIchallenge/blob/master/Info/reflections.pdf) contains an overview of how the program works and an explanation of some of the code in APIchallenge.go. In the Info folder you can also find full sized images of the diagrams in reflections.pdf as well as brief screen recordings demonstrating the features of the UI and the terminal output of the Classic and Timing methods. 
