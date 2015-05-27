package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/virifi/rbuild"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
)

const (
	ExitCodeOK = iota
	ExitCodeError
	ExitCodeParseFlagError
	ExitCodeParseConfigError
	ExitCodeListenError
	ExitCodeInvalidConfigError
	ExitCodeAcceptError
	ExitCodeWorkerError
	ExitCodeParseEnvJsonError
)

var errNoRepositories = errors.New("There are no repositories.")

type CLI struct {
	outStream, errStream io.Writer
}

func (c *CLI) Run(args []string) int {
	flags := flag.NewFlagSet("rbuild-bot", flag.ContinueOnError)
	flags.SetOutput(c.errStream)

	var version bool
	flags.BoolVar(&version, "version", false, "Print version information and quit")

	// Worker options
	var worker bool
	flags.BoolVar(&worker, "worker", false, "Run rbuild-bot as worker")
	var workerRepoName string
	flags.StringVar(&workerRepoName, "repo", "", "Repository name to build [Only effective in worker mode]")
	var workerRepoAbsPath string
	flags.StringVar(&workerRepoAbsPath, "repopath", "", "Repository path to build [Only effective in worker mode]")
	var workerBranch string
	flags.StringVar(&workerBranch, "branch", "", "Branch to build [Only effective in worker mode]")
	var workerCommit string
	flags.StringVar(&workerCommit, "commit", "", "Commit sha1 to build [Only effective in worker mode]")
	var workerEnv string
	flags.StringVar(&workerEnv, "env", "", "EnvItem json [Only effective in worker mode]")

	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeParseFlagError
	}

	if version {
		fmt.Fprintf(c.errStream, "rbuild-bot version %s\n", Version)
		return ExitCodeOK
	}

	if worker {
		var envItems []rbuild.EnvItem
		if err := json.Unmarshal([]byte(workerEnv), &envItems); err != nil {
			fmt.Fprintf(c.errStream, "Parsing env json failed : %v\n", err)
			return ExitCodeParseEnvJsonError
		}
		workerCommands := flags.Args()
		return c.runWorker(workerRepoName, workerRepoAbsPath, workerBranch, workerCommit, envItems, workerCommands)
	}

	if len(flags.Args()) != 1 {
		fmt.Fprintf(c.errStream, "Invalid number of arguments\n")
		fmt.Fprintf(c.errStream, "Usage: %v [options] <config file path>\n", args[0])
		return ExitCodeParseFlagError
	}
	configAbsPath, err := filepath.Abs(flags.Args()[0])
	if err != nil {
		fmt.Fprintf(c.errStream, "Could not get absolute path for config file : %v\n", err)
		return ExitCodeError
	}
	port, repos, err := parseConfigFile(configAbsPath)
	if err != nil {
		fmt.Fprintf(c.errStream, "Parsing config file failed : %v\n", err)
		return ExitCodeParseConfigError
	}

	cmdPath := args[0]

	return c.runServer(port, cmdPath, repos)
}

type Config struct {
	Port         int          `json:"port"`
	Repositories []Repository `json:"repositories"`
}
type Repository struct {
	Name string    `json:"name"`
	Path string    `json:"path"`
	Env  []EnvItem `json:"env"`
}
type EnvItem struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Prepend bool   `json:"prepend"`
}

func parseConfigFile(configAbsPath string) (int, []rbuild.Repository, error) {
	f, err := os.Open(configAbsPath)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var conf Config
	err = dec.Decode(&conf)
	if err != nil {
		return 0, nil, err
	}
	if len(conf.Repositories) == 0 {
		return 0, nil, errNoRepositories
	}

	configDir := filepath.Dir(configAbsPath)
	repos := make([]rbuild.Repository, 0)
	for _, r := range conf.Repositories {
		if len(r.Name) == 0 {
			return 0, nil, fmt.Errorf("name is empty")
		}
		if len(r.Path) == 0 {
			return 0, nil, fmt.Errorf("path is empty")
		}
		absPath := filepath.Join(configDir, r.Path)

		env := make([]rbuild.EnvItem, 0)
		for _, item := range r.Env {
			env = append(env, rbuild.EnvItem{item.Name, item.Value, item.Prepend})
		}
		repos = append(repos, rbuild.Repository{
			Name:    r.Name,
			AbsPath: absPath,
			Env:     env,
		})
	}

	return conf.Port, repos, nil
}

func (c *CLI) runServer(port int, cmdPath string, repos []rbuild.Repository) int {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Fprintf(c.errStream, "listen failed : %v\n", err)
		return ExitCodeListenError
	}
	defer l.Close()

	botSrv, err := rbuild.NewBotServer(cmdPath, repos)
	if err != nil {
		fmt.Fprintf(c.errStream, "new bot server failed : %v\n", err)
		return ExitCodeInvalidConfigError
	}

	log.Println("Bot is running on", l.Addr())
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Fprintf(c.errStream, "accept failed : %v\n", err)
			return ExitCodeAcceptError
		}
		go botSrv.ServeConn(conn)
	}

	return ExitCodeOK
}

func (c *CLI) runWorker(repoName, repoAbsPath, branch, commit string, env []rbuild.EnvItem, commands []string) int {
	if _, err := os.Stat(repoAbsPath); err != nil {
		err := os.MkdirAll(repoAbsPath, 0755)
		if err != nil {
			fmt.Fprintf(c.errStream, "Could not create directory : %v\n", err)
			return ExitCodeError
		}
	}
	if err := os.Chdir(repoAbsPath); err != nil {
		fmt.Fprintf(c.errStream, "Could not change directory : %v\n", err)
		return ExitCodeError
	}

	bw := rbuild.NewBotWorker(c.outStream, c.errStream, env)
	fmt.Fprintf(c.errStream, "Checking out\nrepo : %v\nbranch : %v\ncommit : %v\n", repoName, branch, commit)
	err := bw.Checkout(repoName, branch, commit)
	if err != nil {
		fmt.Fprintf(c.errStream, "Checking out failed : %v\n", err)
		return ExitCodeError
	}

	err = bw.Run(commands[0], commands[1:]...)
	if err != nil {
		fmt.Fprintf(c.errStream, "Worker finished with error : %v\n", err)
		return ExitCodeWorkerError
	}
	return ExitCodeOK
}
