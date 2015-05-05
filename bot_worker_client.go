package rbuild

import (
	"io"
	"os/exec"
)

type BotWorkerClient interface {
	Run(outStream, errStream io.Writer) error
}

func NewBotWorkerClient(botCmdPath, repoName, repoAbsPath, branch, commit, command string) BotWorkerClient {
	cmd := exec.Command(
		botCmdPath, "-worker",
		"-repo", repoName,
		"-repopath", repoAbsPath,
		"-branch", branch,
		"-commit", commit,
		"-command", command)
	return &botWorkerClient{cmd}
}

type botWorkerClient struct {
	cmd *exec.Cmd
}

func (bc *botWorkerClient) Run(outStream, errStream io.Writer) error {
	bc.cmd.Stdout = outStream
	bc.cmd.Stderr = errStream
	return bc.cmd.Run()
}
