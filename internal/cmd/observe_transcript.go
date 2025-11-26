package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/harrison/conductor/internal/behavioral"
	"github.com/spf13/cobra"
)

var observeTranscriptRaw bool

// NewObserveTranscriptCmd creates the 'conductor observe transcript' subcommand
func NewObserveTranscriptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcript <session-id>",
		Short: "Display session transcript",
		Long: `Display a human-readable transcript of a session's events.

Shows assistant messages, user messages, and tool invocations in chronological order
with timestamps and color formatting.

Use --raw to disable colors and emojis for plain text output.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			return DisplaySessionTranscript(sessionID, observeProject, observeTranscriptRaw)
		},
	}

	cmd.Flags().BoolVar(&observeTranscriptRaw, "raw", false, "Plain text output (no colors/emojis)")

	return cmd
}

// DisplaySessionTranscript displays the transcript for a session
func DisplaySessionTranscript(sessionID, project string, raw bool) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}

	// Disable color if raw mode
	if raw {
		color.NoColor = true
	}

	aggregator := behavioral.NewAggregator(50)

	var sessionData *behavioral.SessionData
	var err error

	// If project specified, use it directly
	if project != "" {
		sessions, listErr := aggregator.ListSessions(project)
		if listErr != nil {
			return fmt.Errorf("list sessions: %w", listErr)
		}
		for _, s := range sessions {
			if s.SessionID == sessionID || strings.Contains(s.SessionID, sessionID) {
				sessionData, err = behavioral.ParseSessionFile(s.FilePath)
				if err != nil {
					return fmt.Errorf("parse session file: %w", err)
				}
				break
			}
		}
	} else {
		// Search across all projects
		sessionInfo, _, data, err := findSessionAcrossProjects(aggregator, sessionID)
		if err != nil {
			return fmt.Errorf("session not found: %s", sessionID)
		}
		if data != nil {
			sessionData = data
		} else if sessionInfo != nil {
			sessionData, err = behavioral.ParseSessionFile(sessionInfo.FilePath)
			if err != nil {
				return fmt.Errorf("parse session file: %w", err)
			}
		}
	}

	if sessionData == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if len(sessionData.Events) == 0 {
		fmt.Println("No events in session")
		return nil
	}

	// Sort events by timestamp
	events := sessionData.Events
	sort.Slice(events, func(i, j int) bool {
		return events[i].GetTimestamp().Before(events[j].GetTimestamp())
	})

	// Format transcript
	opts := behavioral.DefaultTranscriptOptions()
	if raw {
		opts.ColorOutput = false
	}
	opts.TruncateLength = 500 // Allow longer text in transcript view

	transcript := behavioral.FormatTranscript(events, opts)
	fmt.Print(transcript)

	return nil
}
