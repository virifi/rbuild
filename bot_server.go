package rbuild

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/hashicorp/yamux"
	"github.com/k0kubun/pp"
	"log"
	"net"
	"os/exec"
	"syscall"
)

type BotServer struct {
	botCmdPath string
	repos      []Repository
}

func NewBotServer(botCmdAbsPath string, repos []Repository) (*BotServer, error) {
	pp.Println(repos)
	return &BotServer{botCmdPath: botCmdAbsPath, repos: repos}, nil
}

type BuildWork struct {
	RepoName string `json:"repo_name"`
	Branch   string `json:"branch"`
	Commit   string `json:"commit"`
	Command  string `json:"command"`
}

func (bs *BotServer) ServeConn(conn net.Conn) {
	log.Println("New connection :", conn.RemoteAddr().String())
	defer func() {
		conn.Close()
		log.Println("Connection closed :", conn.RemoteAddr().String())
	}()

	defConfig := yamux.DefaultConfig()
	defConfig.AcceptBacklog = 1
	session, err := yamux.Server(conn, defConfig)
	if err != nil {
		log.Println("yamux.Server failed :", err)
		return
	}
	defer session.Close()

	// Read build work
	buildWorkStream, err := session.Accept()
	if err != nil {
		log.Println("Accepting buildWorkStream failed :", err)
		return
	}
	defer buildWorkStream.Close()
	dec := json.NewDecoder(buildWorkStream)
	var work BuildWork
	err = dec.Decode(&work)
	if err != nil {
		log.Println("Reading BuildWork failed : ", err)
		return
	}
	repo, ok := bs.findRepo(work.RepoName)
	if !ok {
		log.Println("Could not find repository")
		return
	}

	outStream, err := session.Accept()
	if err != nil {
		log.Println("Accepting outStream failed :", err)
		return
	}
	defer outStream.Close()

	errStream, err := session.Accept()
	if err != nil {
		log.Println("Accepting errStream failed :", err)
		return
	}
	defer errStream.Close()

	statusCodeConn, err := session.Accept()
	if err != nil {
		log.Println("Accepting statusCodeConn failed :", err)
		return
	}
	defer statusCodeConn.Close()

	bc := NewBotWorkerClient(bs.botCmdPath, repo.Name, repo.AbsPath, work.Branch, work.Commit, work.Command)
	err = bc.Run(outStream, errStream)
	exitStatus := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if s, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				log.Println("BotWorker exited with status :", s.ExitStatus())
				exitStatus = s.ExitStatus()
			} else {
				panic(errors.New("Unimplemented for system where exec.ExitError.Sys() is not syscall.WaitStatus."))
			}
		} else {
			log.Println("BotWorker Run failed : ", err)
			exitStatus = 1
		}
	}
	binary.Write(statusCodeConn, binary.BigEndian, int32(exitStatus))
}

func (bs *BotServer) findRepo(repoName string) (Repository, bool) {
	for _, repo := range bs.repos {
		if repo.Name == repoName {
			return repo, true
		}
	}
	return Repository{}, false
}
