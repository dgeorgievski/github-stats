package ghclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// number of Commits and unique commiters
type CommitResults struct {
	Commits          uint16
	Committers       uint16
	BranchesCnt      uint16
	BranchesStaleCnt uint16
	Timestamp        int64
}

// captures Branch commit data
type Branch struct {
	Name   string `json:"name"`
	Commit struct {
		Sha string `json:"sha"`
		URL string `json:"url"`
	}
}

// fragment of the GitHub commits JSOB response
type Commit struct {
	Sha    string `json:"sha"`
	Commit struct {
		Author struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

// get all commits in a repo since now() - interval
func HTTPGitHubAllCommits(url string,
	token string,
	interval time.Duration) (CommitResults, error) {

	var branches []Branch
	var result CommitResults
	var nextUrl = fmt.Sprintf("%s/branches", url)
	var queryId string
	var leftPage, lastPage uint64
	isLink := false

	// fmt.Printf(" > nextUrl[%d]: %s\n", leftPage, nextUrl)

	for {

		req, err := http.NewRequest(
			"GET",
			nextUrl,
			nil)

		if err != nil {
			return result, err
		}

		q := req.URL.Query()
		q.Add("per_page", "100")

		req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return result, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			err = errors.New(
				"resp.Branch.StatusCode: " +
					strconv.Itoa(resp.StatusCode))
			return result, err
		}

		// get the pagination info
		if !isLink {
			isLink = true
			linkRes := parseLink(resp.Header.Get("link"))
			queryId = linkRes["queryId"]
			leftPage, _ = strconv.ParseUint(linkRes["leftPage"], 10, 64)
			lastPage, _ = strconv.ParseUint(linkRes["lastPage"], 10, 64)
		}

		// fmt.Println("link: ", resp.Header.Get("link"))
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return result, err
		}

		json.Unmarshal(contents, &branches)

		// get all commits from all branches since now() - interval
		for _, branch := range branches {
			// res := CommitResults{0, 0, 0}

			branchCommitts, err := HTTPGitHubAllBranchCommits(
				url,
				branch.Name,
				token,
				interval)

			if err != nil {
				return result, err
			}

			result.Commits += branchCommitts.Commits
			result.Committers += branchCommitts.Committers
			result.BranchesCnt++
			result.BranchesStaleCnt += branchCommitts.BranchesStaleCnt
		}

		// stop if last page was reached
		if leftPage == 0 || leftPage > lastPage {
			break
		}

		nextUrl = fmt.Sprintf("https://api.github.com/repositories/%s/branches?page=%d",
			queryId,
			leftPage)

		// fmt.Printf(" > nextUrl[%d]: %s\n", leftPage, nextUrl)
		leftPage++
	}

	result.Timestamp = time.Now().UTC().UnixNano()

	return result, nil
}

// get commits result for a given branch since now() - interval
func HTTPGitHubAllBranchCommits(url string,
	branchName string,
	token string,
	interval time.Duration) (CommitResults, error) {
	var isBranchStale bool = true

	var last3m time.Duration = time.Hour * 2160 // ~ 3 months in hours
	var commits []Commit
	res := CommitResults{0, 0, 0, 0, 0}

	currentTime := time.Now().UTC()
	sinceIntervalDate := currentTime.Add(-interval)
	since3mDate := currentTime.Add(-last3m)
	iso8601Date := fmt.Sprintf("%d-%d-%dT%d:%d:00Z",
		since3mDate.Year(),
		int(since3mDate.Month()),
		since3mDate.Day(),
		since3mDate.Hour(),
		since3mDate.Minute(),
	)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/commits", url),
		nil)

	if err != nil {
		return res, err
	}

	q := req.URL.Query()
	q.Add("sha", branchName)
	q.Add("since", iso8601Date)
	req.URL.RawQuery = q.Encode()

	// fmt.Println(" > Raw query: ", req.URL.String())
	req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = errors.New(
			"resp.Commit.StatusCode: " +
				strconv.Itoa(resp.StatusCode))
		return res, err
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}

	json.Unmarshal(contents, &commits)

	seen := make(map[string]int)
	for _, c := range commits {

		// commit updated within the interval time period
		if c.Commit.Author.Date.After(sinceIntervalDate) {
			res.Commits++
			if _, ok := seen[c.Commit.Author.Name]; !ok {
				seen[c.Commit.Author.Name] = 1
			}
		}

		// commit updated within the active 3 month interval time period
		// fmt.Println("  >> Commit date: ", c.Commit.Author.Date)
		if isBranchStale && c.Commit.Author.Date.After(since3mDate) {
			isBranchStale = false
		}
	}

	res.Committers = uint16(len(seen))
	res.Timestamp = time.Now().Unix()

	if isBranchStale {
		res.BranchesStaleCnt = 1
	}

	// fmt.Println(" >> Is Branch stale?  ", res.BranchesStaleCnt)
	return res, nil
}

// parses GitHub `link` header and returns a list with 3 elems:
// - query id
// - 2nd page index - must be 2
// - last page > 2
func parseLink(link string) map[string]string {
	linkRes := make(map[string]string)
	re := regexp.MustCompile(`(?P<quid>[0-9]+)/branches\?page=(?P<first>[0-9]+)>; rel="next", <.*/(?:[0-9]+)/branches\?page=(?P<last>[0-9]+)>; rel="last"`)
	parts := re.FindAllStringSubmatch(link, -1)

	if len(parts) != 1 {
		return linkRes
	}

	linkRes["queryId"] = parts[0][1]
	linkRes["leftPage"] = parts[0][2]
	linkRes["lastPage"] = parts[0][3]

	fmt.Println(" > Match found for link", linkRes)

	return linkRes
}
