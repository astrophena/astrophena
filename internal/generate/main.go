package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"go.astrophena.name/base/cli"
	"go.astrophena.name/base/request"
)

func main() { cli.Main(cli.AppFunc(run)) }

type repo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"html_url"`
	Archived    bool   `json:"archived"`
	Private     bool   `json:"private"`
}

func run(ctx context.Context) error {
	env := cli.GetEnv(ctx)
	token := env.Getenv("GITHUB_TOKEN")
	// For local testing.
	if token == "" && env.Getenv("CI") != "true" {
		btoken, err := exec.Command("gh", "auth", "token").Output()
		if err != nil {
			return err
		}
		token = strings.TrimSuffix(string(btoken), "\n")
	}

	repos, err := request.Make[[]repo](ctx, request.Params{
		Method: http.MethodGet,
		URL:    "https://api.github.com/users/astrophena/repos",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Scrubber: strings.NewReplacer(token, "[EXPUNGED]"),
	})
	if err != nil {
		return err
	}

	tmplb, err := os.ReadFile("TEMPLATE.md")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.New("TEMPLATE.md was not found; are you at repository root?")
		}
		return err
	}
	tmpl := string(tmplb)

	var sb strings.Builder
	sb.WriteString(tmpl + "\n")

	for _, repo := range repos {
		if repo.Archived || repo.Private || repo.Name == "astrophena" {
			continue
		}
		fmt.Fprintf(&sb, "- [%s](%s) — %s.\n", repo.Name, repo.URL, lowercaseFirst(repo.Description))
	}

	return os.WriteFile("README.md", []byte(sb.String()), 0o644)
}

func lowercaseFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
