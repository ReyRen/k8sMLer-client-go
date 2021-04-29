package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/sevlyar/go-daemon"
	"log"
	"net/http"
	"os"
)

var addr = flag.String("addr", websocketServer, "http service address")
var mode = flag.String("mode", "", "update or not")

func main() {
	var args []string
	flag.Parse()

	args = append(args, "[k8sMLer daemon]")
	if *mode != "" {
		args = append(args, *mode)
	}

	cntxt := &daemon.Context{
		PidFileName: "k8sMLer.pid",
		PidFilePerm: 0644,
		LogFileName: "k8sMLer.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Env:         nil,
		Args:        args,
		Umask:       027,
	}
	d, err := cntxt.Reborn() // like fork
	if err != nil {
		log.Fatalln("Unable to run: ", err)
	}
	if d != nil {
		return // child is ready, return parent
	}

	defer cntxt.Release()

	var mod string
	if len(os.Args) > 1 {
		// update
		mod = os.Args[1]
	}

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("K8sMLer daemon started")

	listen_main(mod)

}

func listen_main(mod string) {
	QUEUELIST = make([]*headNode, 0)

	IP_POOL := make(map[string]bool)
	if mod == MOD_UPDATE {
		Trace.Println("update mode start up, recovery IP-POOL...")

		tmpbyte := make([]byte, 4096)
		file, error := os.OpenFile(".ippool", os.O_RDONLY, 0766)
		if error != nil {
			fmt.Println(error)
		}

		total, err := file.Read(tmpbyte)
		if err != nil {
			Error.Println(err)
		}
		err = json.Unmarshal(tmpbyte[:total], &IP_POOL) // tmpbyte[:total] for error invalid character '\x00' after top-level value
		if err != nil {
			Error.Println(err)
		}

		//validation
		for k, v := range IP_POOL {
			fmt.Println("'''''''''''''''''''''''''''''''''")
			fmt.Println(k, v)
			fmt.Println("'''''''''''''''''''''''''''''''''")
		}
	}

	hub := newHub()
	go hub.run()

	UPDATEMAP = make(map[string][]string)

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		serveWs(hub, writer, request, mod)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		Error.Println("ListenAndServe: ", err)
	}
}

func init() {
	/*file, err := os.OpenFile("k8sMLer.err",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}*/

	Trace = log.New(os.Stdout,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(os.Stdout,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	/*Error = log.New(io.MultiWriter(file, os.Stderr),
	"ERROR: ",
	log.Ldate|log.Ltime|log.Lshortfile)*/
	Error = log.New(os.Stdout,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}
