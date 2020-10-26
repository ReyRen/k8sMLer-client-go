package main

import (
	"flag"
	"github.com/sevlyar/go-daemon"
	"log"
	"net/http"
)

var addr = flag.String("addr", websocketServer, "http service address")

func main() {
	cntxt := &daemon.Context{
		PidFileName: "k8sMLer.pid",
		PidFilePerm: 0644,
		LogFileName: "k8sMLer.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Env:         nil,
		Args:        []string{"[k8sMLer daemon]"},
		Umask:       027,
	}
	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatalln("Unable to run: ", err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("K8sMLer daemon started")

	listen_main()
}

func listen_main() {
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
