package actions

import (
	"context"
	"os"

	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

// CreateInstallationTokenClient uses an installation token to authenticate to the Github API.
func CreateInstallationTokenClient(instToken, baseURL, uploadURL string) (*github.Client, error) {
	// For installation tokens, Github uses a different token type ("token" instead of "bearer")
	tokenType := "token"
	if os.Getenv("GITHUB_TOKEN_TYPE") != "" {
		tokenType = os.Getenv("GITHUB_TOKEN_TYPE")
	}
	t := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: instToken, TokenType: tokenType})
	c := context.Background()
	tc := oauth2.NewClient(c, t)
	if baseURL != "" {
		return github.NewEnterpriseClient(baseURL, uploadURL, tc)
	}
	return github.NewClient(tc), nil
}
