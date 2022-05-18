package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	ghclient "github.com/dgeorgievski/github-stats/clients"
	cfg "github.com/dgeorgievski/github-stats/config"
)

var (
	configFile string
	Trace      *log.Logger
	Info       *log.Logger
	Warning    *log.Logger
	Error      *log.Logger
)

func main() {
	// parse flags
	flag.StringVar(&configFile, "config", "", "Path to the config YAML file")
	flag.Parse()

	if len(configFile) == 0 {
		fmt.Println("Usage: github --config <path-to-config-file>")
		os.Exit(1)
	}

	config, _ := cfg.ParseConfigFile(configFile)
	// fmt.Println("Config file loaded...")

	//logging
	logFile, err := os.OpenFile(config.GitHub.LogFile, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		Error.Fatalln("Failed to open log file ", logFile, " :", err)
	}
	Init(logFile, logFile, logFile, logFile)

	resCommits, err := config.GenerateBranchList()
	if err != nil {
		Error.Fatalln("Commit list error", err)
	}

	reader := bufio.NewReader(os.Stdin)

LOOP:
	for {
		// wait until Telegraf sends empty input on STDIN
		reader.ReadString('\n')

		// TODO: place the commits handling in a function.
		for org, repos := range resCommits {
			for _, r := range repos.Repos {
				// fmt.Printf("  > Repo URL %s\n", r.URL)
				c, err := ghclient.HTTPGitHubAllCommits(r.URL, repos.Token, repos.Interval)
				if err != nil {
					// TODO log to stderr
					continue LOOP
				}

				stat := fmt.Sprintf("github_commits,org=%s,name=%s commits=%di,contributors=%di,branch_cnt=%di,branch_stale_cnt=%di %d",
					org,
					r.Name,
					c.Commits,
					c.Committers,
					c.BranchesCnt,
					c.BranchesStaleCnt,
					c.Timestamp)

				Info.Println(stat) // send to log file as well
				fmt.Println(stat)
			}
		}
	}
}

func Init(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}
