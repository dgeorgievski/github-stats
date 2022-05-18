package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"gopkg.in/yaml.v3"
)

// github:
//	 apiServer:
//   orgs:
//     - name: wiley
// 		 interval: 1h
//       token: ******
//       repos:
//         - do-k8s-helm
//
// Config for GitHub repos and their access
type Config struct {
	GitHub GitHub `yaml:"github"`
}

type GitHub struct {
	ApiServer string      `yaml:"apiServer"`
	Interval  string      `yaml:"interval"`
	LogFile   string      `yaml:"logFile"`
	Orgs      []GitHubOrg `yaml:"orgs"`
}

type GitHubOrg struct {
	Name  string   `yaml:"name"`
	Token string   `yaml:"token"`
	Repos []string `yaml:"repos"`
}

func ParseConfigFile(path string) (*Config, error) {
	cfg := &Config{}

	// TODO fmt.Println("Reading yaml: ", path)

	yfile, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatal(err)
	}

	err2 := yaml.Unmarshal(yfile, &cfg)
	if err2 != nil {
		log.Fatal(err2)
	}

	return cfg, nil
}

type GitHubOrgRepos struct {
	Token    string
	Interval time.Duration
	Repos    []GitHubUrlRepo
}

type GitHubUrlRepo struct {
	Name string
	URL  string
}

// Creates the list of Github API calls for each repo
func (c *Config) GenerateBranchList() (map[string]GitHubOrgRepos, error) {

	var result = make(map[string]GitHubOrgRepos)
	apiServer := c.GitHub.ApiServer
	hr, err := time.ParseDuration(c.GitHub.Interval)
	if err != nil {
		// TODO - stderr; fmt.Println("Invalid duration. Setting to 1h")
		hr, _ = time.ParseDuration("1h")
	}

	for _, org := range c.GitHub.Orgs {
		var rs []GitHubUrlRepo
		for _, repo := range org.Repos {

			url := fmt.Sprintf("%s/repos/%s/%s",
				apiServer,
				org.Name,
				repo)

			rs = append(rs, GitHubUrlRepo{
				Name: repo,
				URL:  url,
			})
		}

		result[org.Name] = GitHubOrgRepos{
			Token:    org.Token,
			Interval: hr,
			Repos:    rs,
		}
	}

	return result, nil
}
