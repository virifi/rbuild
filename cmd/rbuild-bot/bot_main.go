package main

import (
	// "log"
	// "net/http"
	// _ "net/http/pprof"
	"os"
)

const Version string = "v0.1.0"

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	cli := &CLI{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(cli.Run(os.Args))
}
