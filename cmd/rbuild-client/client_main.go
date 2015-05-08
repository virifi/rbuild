package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/yamux"
	"github.com/virifi/rbuild"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 6 {
		fmt.Printf("Usage: %s <bot address> <repository name> <branch> <commit> <command>\n", os.Args[0])
		os.Exit(1)
	}

	botAddr := os.Args[1]
	repoName := os.Args[2]
	branch := os.Args[3]
	commit := os.Args[4]
	commands := os.Args[5:]

	conn, err := net.Dial("tcp", botAddr)
	if err != nil {
		log.Println("Connecting to bot failed :", err)
		os.Exit(1)
	}
	defer conn.Close()

	defConfig := yamux.DefaultConfig()
	defConfig.AcceptBacklog = 1
	session, err := yamux.Client(conn, defConfig)
	if err != nil {
		log.Println("yamux.Client failed :", err)
		os.Exit(1)
	}
	defer session.Close()

	buildWorkStream, err := session.Open()
	if err != nil {
		log.Println("Could not open buildWorkStream :", err)
		os.Exit(1)
	}
	defer buildWorkStream.Close()

	work := rbuild.BuildWork{
		RepoName: repoName,
		Branch:   branch,
		Commit:   commit,
		Commands: commands,
	}
	enc := json.NewEncoder(buildWorkStream)
	err = enc.Encode(work)
	if err != nil {
		log.Println("Could not write BuildWork :", err)
		os.Exit(1)
	}

	outStream, err := session.Open()
	if err != nil {
		log.Println("Could not open outStream :", err)
		os.Exit(1)
	}
	defer outStream.Close()

	errStream, err := session.Open()
	if err != nil {
		log.Println("Could not open errStream :", err)
		os.Exit(1)
	}
	defer errStream.Close()

	exitStatusStream, err := session.Open()
	if err != nil {
		log.Println("Could not open exitStatusStream :", err)
		os.Exit(1)
	}
	defer exitStatusStream.Close()

	go io.Copy(os.Stdout, outStream)
	go io.Copy(os.Stderr, errStream)
	var exitStatus int32
	err = binary.Read(exitStatusStream, binary.BigEndian, &exitStatus)
	if err != nil {
		log.Println("Reading exit status failed :", err)
		os.Exit(1)
	}
	os.Exit(int(exitStatus))
}
