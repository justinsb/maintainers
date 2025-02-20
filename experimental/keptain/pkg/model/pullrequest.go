package model

import "fmt"

// PullRequest is a pull request
type PullRequest struct {
	// The number of the PR
	Number int

	// The title of the PR
	Title string

	// The author of the PR
	Author string

	// The SHA of the PR head commit
	CommitSHA string

	// The SHA of the PR base commit (the branch we're merging into)
	BaseSHA string

	// Files is a list of files that were changed in the PR
	Files []string
}

func (p *PullRequest) Link() string {
	return fmt.Sprintf("https://github.com/kubernetes/enhancements/pull/%d", p.Number)
}
