// Copyright (c) 2020, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/mod/modfile"
	"golang.org/x/oauth2"
)

func main() {
	if err := mainErr(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	flagCount   = flag.Int("count", 100, "number of modules to output")
	flagVerbose = flag.Bool("v", false, "print verbose log output")
)

func vlogf(format string, args ...interface{}) {
	if *flagVerbose {
		log.Printf(format, args...)
	}
}

type repository struct {
	URL githubv4.URI

	StargazerCount githubv4.Int
	ForkCount      githubv4.Int

	DefaultBranchRef struct {
		Target struct {
			Commit struct {
				OID        githubv4.String
				PushedDate githubv4.DateTime
			} `graphql:"... on Commit"`
		}
	}

	GoModObj struct {
		Blob struct {
			Text githubv4.String
		} `graphql:"... on Blob"`
	} `graphql:"object(expression: \"HEAD:go.mod\")"`
}

type repoSearch struct {
	Search struct {
		RepositoryCount int
		PageInfo        struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
		Nodes []struct {
			Repository repository `graphql:"... on Repository"`
		}
	} `graphql:"search(first: 100, after: $repocursor, type: REPOSITORY, query: $querystring)"`
	RateLimit struct {
		Cost      githubv4.Int
		Limit     githubv4.Int
		Remaining githubv4.Int
		ResetAt   githubv4.DateTime
	}
}

type moduleStats struct {
	modulePath string
	sourceURL  string // e.g. github https URL
	version    string // e.g. commit ID
	score      int
}

func mainErr() error {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: corpus [flags]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(flag.Args()) > 0 {
		flag.Usage() // we don't take any args just yet
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewClient(httpClient)
	ctx := context.Background() // TODO: ^C handling, global timeout

	oneYearAgo := time.Now().UTC().Add(-time.Hour * 24 * 364) // 364 days
	queryString := fmt.Sprintf(`archived:false is:public stars:>=100 pushed:>=%s language:go sort:stars`,
		oneYearAgo.Format("2006-01-02"))
	vlogf("query string: %q", queryString)

	var cursor *githubv4.String
	var modules []moduleStats
moreLoop:
	for page := 1; ; page++ {
		if cursor == nil {
			vlogf("querying first page of results")
		} else {
			vlogf("%d/%d done; querying page %d with cursor %s", len(modules), *flagCount, page, *cursor)
		}

		// GraphQL queries can take a few seconds.
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		var query repoSearch
		variables := map[string]interface{}{
			"repocursor":  cursor,
			"querystring": githubv4.String(queryString),
		}
		if err := client.Query(ctx, &query, variables); err != nil {
			return err
		}
		cancel() // we are in a loop

		for _, node := range query.Search.Nodes {
			repo := node.Repository
			module := moduleStats{
				modulePath: modfile.ModulePath([]byte(repo.GoModObj.Blob.Text)),
				sourceURL:  repo.URL.String(),
			}
			if module.modulePath == "" {
				vlogf("no module found in %s; skipping", module.sourceURL)
				continue
			}

			lastCommit := repo.DefaultBranchRef.Target.Commit
			if lastCommit.PushedDate.Before(oneYearAgo) {
				// Unfortunately, github's "pushed" filter is too broad,
				// as it includes non-default branches.
				vlogf("no recent pushes to %s; skipping", module.sourceURL)
				continue
			}
			module.version = string(lastCommit.OID)

			// For now, the score is just the sum of stars and forks.
			module.score = int(repo.StargazerCount + repo.ForkCount)

			modules = append(modules, module)
			if len(modules) >= *flagCount {
				break moreLoop
			}
		}
		if !query.Search.PageInfo.HasNextPage {
			return fmt.Errorf("ran out of repositories in search after %d pages; found %d modules", page, len(modules))
		}
		cursor = &query.Search.PageInfo.EndCursor
	}

	// TODO: Since we sort after fetching, and we only sort the query by
	// stars while our score also counts forks, we might skew the results
	// towards the cutoff. Perhaps query for an extra 20%, sort, then
	// discard the tail end.
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].score > modules[j].score
	})

	w := csv.NewWriter(os.Stdout)
	w.Comma = '\t'
	w.Write([]string{"module", "version", "source", "score"})
	for _, m := range modules {
		w.Write([]string{
			m.modulePath,
			m.version,
			m.sourceURL,
			strconv.Itoa(m.score),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	return nil
}
