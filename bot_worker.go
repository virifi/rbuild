package rbuild

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type BotWorker struct {
	outStream io.Writer
	errStream io.Writer
}

func NewBotWorker(outStream, errStream io.Writer) *BotWorker {
	return &BotWorker{outStream: outStream, errStream: errStream}
}

func (bw *BotWorker) Checkout(repoName, branch, commit string) error {
	log.Println("BotWorker#Checkout")
	if !bw.cwdIsGitDir() {
		err := bw.cloneRepo(repoName)
		if err != nil {
			return err
		}
	}
	if err := bw.fetch(); err != nil {
		return err
	}
	if err := bw.checkoutCommit(commit); err != nil {
		return err
	}
	return nil
}

func (bw *BotWorker) Run(command string) error {
	log.Println("BotWorker#Run")
	return bw.runCommand(command)
}

func (bw *BotWorker) cwdIsGitDir() bool {
	_, err := os.Stat(".git")
	return err == nil
}

func (bw *BotWorker) cloneRepo(repoName string) error {
	url := fmt.Sprintf("git@github.com:%s.git", repoName)
	return bw.runCommand("git", "clone", url, ".")
}

func (bw *BotWorker) fetch() error {
	return bw.runCommand("git", "fetch")
}

func (bw *BotWorker) checkoutCommit(commitSha1 string) error {
	return bw.runCommand("git", "checkout", commitSha1)
}

func (bw *BotWorker) runCommand(name string, args ...string) error {
	fmt.Fprintf(bw.outStream, "Running command %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = bw.outStream
	cmd.Stderr = bw.errStream
	return cmd.Run()
}
