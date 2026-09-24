package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
)

// rawMethods is the set of HTTP methods the raw passthrough accepts.
var rawMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodDelete: true,
}

func (a *App) newRawCmd() *cobra.Command {
	var queryPairs []string
	cmd := &cobra.Command{
		Use:   "raw <method> <path>",
		Short: "Make a raw Trello API request",
		Long: `Make an untyped request to the Trello API and print the raw JSON
response.

<method> is one of GET, POST, PUT, or DELETE. <path> is the API path
starting with "/", e.g. /members/me. The key and token are injected
automatically. Use -q/--query to add query parameters (repeatable).`,
		Example: `  trello raw GET /members/me
  trello raw GET /boards/5abbe4b7ddc1b351ef961414/lists --query fields=id,name
  trello raw POST /cards --query name="New card" --query idList=5abbe4b7ddc1b351ef961415 --json`,
		Args: exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			if !rawMethods[method] {
				return output.WithCode(fmt.Errorf("invalid method %q: must be one of GET, POST, PUT, DELETE", args[0]), output.ExitUsage)
			}
			path := args[1]
			if !strings.HasPrefix(path, "/") {
				return output.WithCode(fmt.Errorf("invalid path %q: must start with /", path), output.ExitUsage)
			}
			query := url.Values{}
			for _, kv := range queryPairs {
				k, v, found := strings.Cut(kv, "=")
				if !found {
					return output.WithCode(fmt.Errorf("invalid --query %q: expected k=v", kv), output.ExitUsage)
				}
				query.Add(k, v)
			}
			return a.runRaw(cmd.Context(), method, path, query)
		},
	}
	cmd.Flags().StringArrayVarP(&queryPairs, "query", "q", nil, "query parameter k=v (repeatable)")
	return cmd
}

func (a *App) runRaw(ctx context.Context, method, path string, query url.Values) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	var raw json.RawMessage
	if err := client.Do(ctx, method, path, query, nil, &raw); err != nil {
		return err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	out := a.out()
	if out.JSON {
		b := append([]byte(nil), raw...)
		if b[len(b)-1] != '\n' {
			b = append(b, '\n')
		}
		_, err := out.Out.Write(b)
		return err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return err
	}
	buf.WriteByte('\n')
	_, err = out.Out.Write(buf.Bytes())
	return err
}
