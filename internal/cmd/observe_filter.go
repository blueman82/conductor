package cmd

import (
	"fmt"

	"github.com/harrison/conductor/internal/behavioral"
)

// ParseFilterFlags converts global observe flags into FilterCriteria
func ParseFilterFlags() (behavioral.FilterCriteria, error) {
	criteria := behavioral.FilterCriteria{
		Search:     observeSession, // Using session flag for general search
		EventType:  observeFilterType,
		ErrorsOnly: observeErrorsOnly,
	}

	// Parse time range if provided
	if observeTimeRange != "" {
		since, err := behavioral.ParseTimeRange(observeTimeRange)
		if err != nil {
			return criteria, fmt.Errorf("invalid time range: %w", err)
		}
		criteria.Since = since
	}

	// Validate criteria
	if err := criteria.Validate(); err != nil {
		return criteria, err
	}

	return criteria, nil
}

// ValidateFilterTimeRange validates the time range flag
func ValidateFilterTimeRange() error {
	if observeTimeRange == "" {
		return nil
	}
	return behavioral.ValidateTimeRange(observeTimeRange)
}

// BuildFilterDescription returns a human-readable description of active filters
func BuildFilterDescription(criteria behavioral.FilterCriteria) string {
	if (criteria == behavioral.FilterCriteria{}) {
		return "No filters applied"
	}

	desc := "Filters: "
	parts := []string{}

	if criteria.Search != "" {
		parts = append(parts, fmt.Sprintf("search='%s'", criteria.Search))
	}
	if criteria.EventType != "" {
		parts = append(parts, fmt.Sprintf("type='%s'", criteria.EventType))
	}
	if criteria.ErrorsOnly {
		parts = append(parts, "errors-only")
	}
	if !criteria.Since.IsZero() {
		parts = append(parts, fmt.Sprintf("since='%s'", criteria.Since.Format("2006-01-02 15:04:05")))
	}
	if !criteria.Until.IsZero() {
		parts = append(parts, fmt.Sprintf("until='%s'", criteria.Until.Format("2006-01-02 15:04:05")))
	}

	if len(parts) == 0 {
		return "No filters applied"
	}

	for i, part := range parts {
		if i > 0 {
			desc += ", "
		}
		desc += part
	}

	return desc
}
