package store

import (
	"context"
	"fmt"
	"maps"
	"slices"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v69/github"
	"k8s.io/klog/v2"
	"sigs.k8s.io/maintainers/experiments/keptain/pkg/model"
)

func (r *Repository) indexPullRequests(ctx context.Context) error {
	githubClient := github.NewClient(nil)
	githubPullRequests, _, err := githubClient.PullRequests.List(ctx, "kubernetes", "enhancements", nil)
	if err != nil {
		return fmt.Errorf("error listing pull requests: %w", err)
	}

	var prs []*model.PullRequest

	for _, githubPullRequest := range githubPullRequests {
		prs = append(prs, &model.PullRequest{
			Number:    githubPullRequest.GetNumber(),
			Title:     githubPullRequest.GetTitle(),
			Author:    githubPullRequest.GetUser().GetLogin(),
			CommitSHA: githubPullRequest.GetHead().GetSHA(),
			BaseSHA:   githubPullRequest.GetBase().GetSHA(),
		})
	}

	// For each PR, we need a list of commits, and changes files.
	// This lets us associate a PR with a KEP.
	gitRepo, err := gogit.PlainOpen(r.basePath)
	if err != nil {
		return fmt.Errorf("error opening git repository: %w", err)
	}

	for _, pr := range prs {
		klog.Infof("indexing PR %d", pr.Number)

		headCommit, err := gitRepo.CommitObject(plumbing.NewHash(pr.CommitSHA))
		if err != nil {
			return fmt.Errorf("error getting head commit %q for PR %d: %w", pr.CommitSHA, pr.Number, err)
		}

		baseCommit, err := gitRepo.CommitObject(plumbing.NewHash(pr.BaseSHA))
		if err != nil {
			return fmt.Errorf("error getting base commit %q for PR %d: %w", pr.BaseSHA, pr.Number, err)
		}

		mergeBase, err := headCommit.MergeBase(baseCommit)
		if err != nil {
			return fmt.Errorf("error getting merge base for PR %d: %w", pr.Number, err)
		}
		if len(mergeBase) == 0 {
			return fmt.Errorf("no merge base found for PR %d", pr.Number)
		}
		if len(mergeBase) > 1 {
			return fmt.Errorf("multiple merge bases found for PR %d", pr.Number)
		}

		commitIter, err := gitRepo.Log(&gogit.LogOptions{
			From: plumbing.NewHash(pr.CommitSHA),
		})
		if err != nil {
			return fmt.Errorf("reading commits in PR %d from %s: %w", pr.Number, pr.CommitSHA, err)
		}

		var commits []*object.Commit
		stopAt := mergeBase[0].Hash
		for {
			commit, err := commitIter.Next()
			if err != nil {
				klog.Infof("mergeBase is %v", stopAt)
				return fmt.Errorf("walking commit log for PR %d from %s: %w", pr.Number, pr.CommitSHA, err)
			}
			if commit.Hash == stopAt {
				break
			}
			commits = append(commits, commit)
		}

		baseTree, err := commits[len(commits)-1].Tree()
		if err != nil {
			return fmt.Errorf("error getting base tree for PR %d: %w", pr.Number, err)
		}
		headTree, err := commits[0].Tree()
		if err != nil {
			return fmt.Errorf("error getting head tree for PR %d: %w", pr.Number, err)
		}

		changes, err := baseTree.Diff(headTree)
		if err != nil {
			return fmt.Errorf("error getting diff for PR %d: %w", pr.Number, err)
		}
		files := make(map[string]bool)
		for _, change := range changes {
			if change.To.Name != "" {
				files[change.To.Name] = true
			}
			if change.From.Name != "" {
				files[change.From.Name] = true
			}
		}

		pr.Files = slices.Sorted(maps.Keys(files))

		// TODO: Get the KEP path from the PR files.
	}
	r.pullRequests = prs

	return nil
}

// ListPullRequests returns a list of all pull requests in the repository.
func (r *Repository) ListPullRequests() ([]*model.PullRequest, error) {
	return r.pullRequests, nil
}
