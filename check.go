package resource

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/telia-oss/github-pr-resource/pullrequest"
)

func findPulls(since time.Time, gh Github) ([]pullrequest.PullRequest, error) {
	if since.IsZero() {
		since = time.Now().AddDate(-3, 0, 0)
	}
	return gh.ListOpenPullRequests(since)
}

// Check (business logic)
func Check(request CheckRequest, manager Github) (CheckResponse, error) {
	var response CheckResponse

	pulls, err := findPulls(request.Version.UpdatedDate, manager)
	if err != nil {
		return nil, fmt.Errorf("failed to get last commits: %s", err)
	}

	paths := request.Source.Paths
	iPaths := request.Source.IgnorePaths

	log.Println("total pulls found:", len(pulls))

	for _, p := range pulls {
		log.Printf("evaluate pull: %+v\n", p)
		if !newVersion(request, p) {
			log.Println("no new version found")
			continue
		}

		if len(paths)+len(iPaths) > 0 {
			log.Println("pattern/s configured")
			p.Files, err = compareVersionsChangedFiles(p, request, manager)

			if err != nil {
				log.Println("couldn't get version: ")
				return nil, err
			}

			log.Println("paths configured:", paths)
			log.Println("ignore paths configured:", iPaths)
			log.Println("changed files found:", p.Files)

			switch {
			// if `paths` is configured && NONE of the changed files match `paths` pattern/s
			case pullrequest.Patterns(paths)(p) && !pullrequest.Files(paths, false)(p):
				log.Println("paths excluded pull")
				continue
			// if `ignore_paths` is configured && ALL of the changed files match `ignore_paths` pattern/s
			case pullrequest.Patterns(iPaths)(p) && pullrequest.Files(iPaths, true)(p):
				log.Println("ignore paths excluded pull")
				continue
			}
		}

		response = append(response, NewVersion(p))
	}

	// Sort the commits by date
	sort.Sort(response)

	// If there are no new but an old version = return the old
	if len(response) == 0 && request.Version.PR != 0 {
		log.Println("no new versions, use old")
		response = append(response, request.Version)
	}

	// If there are new versions and no previous = return just the latest
	if len(response) != 0 && request.Version.PR == 0 {
		response = CheckResponse{response[len(response)-1]}
	}

	log.Println("version count in response:", len(response))
	log.Println("versions:", response)

	return response, nil
}

func newVersion(r CheckRequest, p pullrequest.PullRequest) bool {
	switch {
	// negative filters
	case pullrequest.SkipCI(r.Source.DisableCISkip)(p),
		pullrequest.BaseBranch(r.Source.BaseBranch)(p),
		pullrequest.ApprovedReviewCount(r.Source.RequiredReviewApprovals)(p),
		pullrequest.Labels(r.Source.Labels)(p),
		pullrequest.Fork(r.Source.DisableForks)(p):
		return false
	// positive filters
	case pullrequest.Created(r.Version.UpdatedDate)(p),
		pullrequest.BaseRefChanged()(p),
		pullrequest.BaseRefForcePushed()(p),
		pullrequest.HeadRefForcePushed()(p),
		pullrequest.Reopened()(p),
		pullrequest.BuildCI()(p),
		pullrequest.NewCommits(r.Version.UpdatedDate)(p):
		return true
	}

	return false
}

func pullRequestFiles(n int, manager Github) ([]string, error) {
	files, err := manager.GetChangedFiles(n)
	if err != nil {
		return nil, fmt.Errorf("failed to list modified files: %s", err)
	}

	return files, nil
}

func commitChangedFiles(sha string, manager Github) ([]string, error) {
	files, err := manager.GetCommitChangedFiles(sha)
	if err != nil {
		return nil, fmt.Errorf("failed to list modified files: %s", err)
	}

	return files, nil
}

// If there are new versions and no previous
// Get all changed files associated with the PR
// Else get the changed files on a PRs latest commit
func compareVersionsChangedFiles(pr pullrequest.PullRequest, requestPR CheckRequest, manager Github) ([]string, error) {
	files := make([]string, 0)
	var err error

	if (requestPR.Version.PR == 0) || (pr.Number != requestPR.Version.PR) {
		files, err = pullRequestFiles(pr.Number, manager)
		if err != nil {
			return nil, err
		}
	} else {
		pr, err := manager.GetPullRequest(pr.Number, pr.HeadRef.OID)
		if err != nil {
			log.Println("couldn't get pr...")
			return nil, err
		}

		for commitIndex, commit := range pr.Commits {
			if commit.OID == requestPR.Version.Commit {
				for i := commitIndex + 1; i < len(pr.Commits); i++ {

					f, err := commitChangedFiles(pr.Commits[i].OID, manager)
					if err != nil {
						return nil, err
					}
					files = append(files, f...)
				}
				break
			}
		}
	}

	return files, err
}

// CheckRequest ...
type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

// CheckResponse ...
type CheckResponse []Version

func (r CheckResponse) Len() int {
	return len(r)
}

func (r CheckResponse) Less(i, j int) bool {
	return r[j].UpdatedDate.After(r[i].UpdatedDate)
}

func (r CheckResponse) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
