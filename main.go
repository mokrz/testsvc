package main

import (
	"fmt"
	"net/http"
)

func TestMe(n int) int {
	return n * 2
}

func helloHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Hello world!")
	return
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	serveErr := http.ListenAndServe(":8080", nil)

	if serveErr != nil {
		fmt.Printf("failed to ListenAndServe with error: " + serveErr.Error())
	}

	return
}
