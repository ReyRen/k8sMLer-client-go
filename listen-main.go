package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", "172.18.29.80:8066", "http service address")

func main() {
	flag.Parse()
	hub := newHub()
	go hub.run()

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		serveWs(hub, writer, request)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
