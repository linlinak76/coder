package cliui

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/briandowns/spinner"

	"github.com/coder/coder/codersdk"
)

type GitAuthOptions struct {
	Fetch         func(context.Context) ([]codersdk.TemplateVersionGitAuth, error)
	FetchInterval time.Duration
}

func GitAuth(ctx context.Context, writer io.Writer, opts GitAuthOptions) error {
	if opts.FetchInterval == 0 {
		opts.FetchInterval = 500 * time.Millisecond
	}
	gitAuth, err := opts.Fetch(ctx)
	if err != nil {
		return err
	}

	spin := spinner.New(spinner.CharSets[78], 100*time.Millisecond, spinner.WithColor("fgHiGreen"))
	spin.Writer = writer
	spin.ForceOutput = true
	spin.Suffix = " Waiting for Git authentication..."
	defer spin.Stop()

	ticker := time.NewTicker(opts.FetchInterval)
	defer ticker.Stop()
	for _, auth := range gitAuth {
		if auth.Authenticated {
			return nil
		}

		_, _ = fmt.Fprintf(writer, "You must authenticate with %s to create a workspace with this template. Visit:\n\n\t%s\n\n", auth.Type.Pretty(), auth.AuthenticateURL)

		ticker.Reset(opts.FetchInterval)
		spin.Start()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
			gitAuth, err := opts.Fetch(ctx)
			if err != nil {
				return err
			}
			var authed bool
			for _, a := range gitAuth {
				if !a.Authenticated || a.ID != auth.ID {
					continue
				}
				authed = true
				break
			}
			// The user authenticated with the provider!
			if authed {
				break
			}
		}
		spin.Stop()
		_, _ = fmt.Fprintf(writer, "Successfully authenticated with %s!\n\n", auth.Type.Pretty())
	}
	return nil
}
