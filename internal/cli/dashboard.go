package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/PatrickFanella/get-rich-quick/internal/cli/tui"
	"github.com/spf13/cobra"
)

func (s *rootState) newDashboardCommand() *cobra.Command {
	var once bool
	var width int
	var height int

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch the terminal dashboard",
		Long:  "Launch a Bubble Tea dashboard for monitoring portfolio, strategies, risk, configuration, and pipeline activity.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := s.client()
			if err != nil {
				return err
			}

			snapshot, err := s.tuiSnapshot(cmd.Context(), client)
			if err != nil {
				return err
			}

			if once {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), tui.Render(snapshot, width, height))
				return err
			}

			return tui.Run(cmd.Context(), tui.Options{
				Snapshot: snapshot,
				Output:   cmd.OutOrStdout(),
				Width:    width,
				Height:   height,
				Connect: func(ctx context.Context) (tui.EventSource, error) {
					wsURL, err := websocketURL(s.apiURL)
					if err != nil {
						return nil, err
					}

					headers := http.Header{}
					if s.token != "" {
						headers.Set("Authorization", "Bearer "+s.token)
					}
					if s.apiKey != "" {
						headers.Set("X-API-Key", s.apiKey)
					}

					return tui.ConnectWebSocket(ctx, wsURL, headers)
				},
			})
		},
	}

	cmd.Flags().BoolVar(&once, "once", false, "Render a single TUI frame and exit")
	cmd.Flags().IntVar(&width, "width", 120, "Render width for the terminal dashboard")
	cmd.Flags().IntVar(&height, "height", 34, "Render height for the terminal dashboard")

	return cmd
}

func websocketURL(apiURL string) (string, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("invalid api url: %w", err)
	}

	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported api url scheme %q", parsed.Scheme)
	}

	parsed.Path = "/ws"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
