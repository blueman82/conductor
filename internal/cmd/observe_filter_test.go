package cmd

import (
	"testing"
	"time"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/stretchr/testify/assert"
)

func TestParseFilterFlags(t *testing.T) {
	resetObserveFlags := func() {
		observeProject = ""
		observeSession = ""
		observeFilterType = ""
		observeErrorsOnly = false
		observeTimeRange = ""
	}

	t.Run("empty filters", func(t *testing.T) {
		resetObserveFlags()

		criteria, err := ParseFilterFlags()
		assert.NoError(t, err)
		assert.Equal(t, "", criteria.Search)
		assert.Equal(t, "", criteria.EventType)
		assert.False(t, criteria.ErrorsOnly)
		assert.True(t, criteria.Since.IsZero())
	})

	t.Run("with session search", func(t *testing.T) {
		resetObserveFlags()
		observeSession = "test-session"

		criteria, err := ParseFilterFlags()
		assert.NoError(t, err)
		assert.Equal(t, "test-session", criteria.Search)
	})

	t.Run("with filter type", func(t *testing.T) {
		resetObserveFlags()
		observeFilterType = "tool"

		criteria, err := ParseFilterFlags()
		assert.NoError(t, err)
		assert.Equal(t, "tool", criteria.EventType)
	})

	t.Run("with errors only", func(t *testing.T) {
		resetObserveFlags()
		observeErrorsOnly = true

		criteria, err := ParseFilterFlags()
		assert.NoError(t, err)
		assert.True(t, criteria.ErrorsOnly)
	})

	t.Run("with time range", func(t *testing.T) {
		resetObserveFlags()
		observeTimeRange = "24h"

		criteria, err := ParseFilterFlags()
		assert.NoError(t, err)
		assert.False(t, criteria.Since.IsZero())
		assert.True(t, criteria.Since.Before(time.Now()))
	})

	t.Run("with invalid time range", func(t *testing.T) {
		resetObserveFlags()
		observeTimeRange = "invalid"

		_, err := ParseFilterFlags()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time range")
	})

	t.Run("combined filters", func(t *testing.T) {
		resetObserveFlags()
		observeSession = "my-session"
		observeFilterType = "bash"
		observeErrorsOnly = true
		observeTimeRange = "7d"

		criteria, err := ParseFilterFlags()
		assert.NoError(t, err)
		assert.Equal(t, "my-session", criteria.Search)
		assert.Equal(t, "bash", criteria.EventType)
		assert.True(t, criteria.ErrorsOnly)
		assert.False(t, criteria.Since.IsZero())
	})
}

func TestBuildFilterDescription(t *testing.T) {
	t.Run("no filters", func(t *testing.T) {
		criteria := behavioral.FilterCriteria{}
		desc := BuildFilterDescription(criteria)
		assert.Equal(t, "No filters applied", desc)
	})

	t.Run("with search", func(t *testing.T) {
		criteria := behavioral.FilterCriteria{
			Search: "test",
		}
		desc := BuildFilterDescription(criteria)
		assert.Contains(t, desc, "search='test'")
	})

	t.Run("with type", func(t *testing.T) {
		criteria := behavioral.FilterCriteria{
			EventType: "tool",
		}
		desc := BuildFilterDescription(criteria)
		assert.Contains(t, desc, "type='tool'")
	})

	t.Run("with errors only", func(t *testing.T) {
		criteria := behavioral.FilterCriteria{
			ErrorsOnly: true,
		}
		desc := BuildFilterDescription(criteria)
		assert.Contains(t, desc, "errors-only")
	})

	t.Run("combined filters", func(t *testing.T) {
		now := time.Now()
		criteria := behavioral.FilterCriteria{
			Search:     "session-123",
			EventType:  "bash",
			ErrorsOnly: true,
			Since:      now.Add(-7 * 24 * time.Hour),
		}
		desc := BuildFilterDescription(criteria)
		assert.Contains(t, desc, "search='session-123'")
		assert.Contains(t, desc, "type='bash'")
		assert.Contains(t, desc, "errors-only")
		assert.Contains(t, desc, "since=")
	})
}
