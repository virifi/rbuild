package rbuild

import (
	"encoding/json"
	"io"
	"os/exec"
)

type BotWorkerClient interface {
	Run(outStream, errStream io.Writer) error
}

func NewBotWorkerClient(botCmdPath, repoName, repoAbsPath, branch, commit string, commands []string, env []EnvItem) (BotWorkerClient, error) {
	envJson, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	var args []string
	args = append(args,
		"-worker",
		"-repo", repoName,
		"-repopath", repoAbsPath,
		"-branch", branch,
		"-commit", commit,
		"-env", string(envJson))
	args = append(args, commands...)
	cmd := exec.Command(botCmdPath, args...)
	return &botWorkerClient{cmd}, nil
}

type botWorkerClient struct {
	cmd *exec.Cmd
}

func (bc *botWorkerClient) Run(outStream, errStream io.Writer) error {
	bc.cmd.Stdout = outStream
	bc.cmd.Stderr = errStream
	return bc.cmd.Run()
}
