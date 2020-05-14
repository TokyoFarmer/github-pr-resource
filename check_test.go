package resource_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	resource "github.com/telia-oss/github-pr-resource"
	"github.com/telia-oss/github-pr-resource/fakes"
	"github.com/telia-oss/github-pr-resource/pullrequest"
)

var (
	testPullRequests = []pullrequest.PullRequest{}
)

func TestCheck(t *testing.T) {
	tests := []struct {
		description  string
		source       resource.Source
		version      resource.Version
		files        [][]string
		pullRequests []pullrequest.PullRequest
		expected     resource.CheckResponse
	}{}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			github := new(fakes.FakeGithub)

			github.ListOpenPullRequestsReturns(tc.pullRequests, nil)

			for i, file := range tc.files {
				github.GetChangedFilesReturnsOnCall(i, file, nil)
			}

			input := resource.CheckRequest{Source: tc.source, Version: tc.version}
			output, err := resource.Check(input, github)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, output)
			}
			assert.Equal(t, 1, github.ListOpenPullRequestsCallCount())
		})
	}
}
