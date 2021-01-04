# corpus

A corpus of popular Go modules. See [top-100.tsv](top-100.tsv) for the latest sample.

This corpus is to be used when analyzing or studying Go code. For example, when
one wants to change the Go language and estimate how much existing code would
need to be adapted.

For now, this repository simply contains a table with module information,
including where to find the source code and what precise version was recorded.
Downloading all the source code is an exercise left to the user, but will likely
be provided as part of the program soon. Until then, try `go get -d
module-path@version` in a loop.

### Quickstart

Set up a [github access token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/)
with the `public_repo` permission, and run:

```sh
export GITHUB_TOKEN=...
go run . >output.tsv
```

### FAQ

> Measuring popularity is a bit pointless.

Completely agreed. This is just an honest estimation for the purposes of
research. It should not be used as a "top 100 best Go modules" leaderboard.

> Can't the score be gamed?

In practice, not really. We stick to metrics which require manual work; for
example, starring or forking a GitHub repository requires creating an account.
You would need to fake that process tens of thousands of times, which likely
goes against the site's terms of use.

> This list is too GitHub-centric.

I'd love to extend it, for example with gitlab.com and any other popular code
hosting sites which have useful statistics like stars/forks. If you know of any
sites which qualify and are not yet in the issue tracker, please file an issue.

> My project is popular yet it isn't listed.

Note that a Go project must be a Go module and mainly contain Go code in order
to be matched by the code hosting site searches. The popularity score is also an
estimation, not an objective metric.

If you still think there is a bug in the code, please file a bug.
