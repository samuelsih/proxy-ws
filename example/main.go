package main

import "net/http"

func main() {
	println("Starting client")
	http.Handle("/", http.FileServer(http.Dir("./")))
	err := http.ListenAndServe(":16969", nil)
	if err != nil {
		println(err.Error())
	}
}
