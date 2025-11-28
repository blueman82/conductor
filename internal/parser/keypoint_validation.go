package parser

import (
	"fmt"

	"github.com/harrison/conductor/internal/models"
)

const (
	// CriteriaAlignmentWarn emits warnings but does not fail validation
	CriteriaAlignmentWarn = "warn"
	// CriteriaAlignmentStrict treats alignment issues as validation errors
	CriteriaAlignmentStrict = "strict"
	// CriteriaAlignmentOff disables alignment checks
	CriteriaAlignmentOff = "off"
)

// ValidateKeyPointCriteriaAlignment checks that implementation key_points have corresponding
// success/integration criteria. Returns warnings (or errors in strict mode) when the number
// of key_points exceeds the number of criteria the QC system will verify.
func ValidateKeyPointCriteriaAlignment(tasks []models.Task, mode string) (warnings []string, errors []string) {
	if mode == "" {
		mode = CriteriaAlignmentWarn
	}

	if mode == CriteriaAlignmentOff {
		return nil, nil
	}

	for _, task := range tasks {
		keyPointCount := len(task.KeyPoints)
		if keyPointCount == 0 {
			continue
		}

		criteriaCount := len(task.SuccessCriteria)
		if task.Type == "integration" {
			criteriaCount += len(task.IntegrationCriteria)
		}

		if keyPointCount > criteriaCount {
			msg := fmt.Sprintf("Task %s (%s): %d key_points but only %d success_criteria/integration_criteria; %d key_point(s) may not be explicitly verified",
				task.Number, task.Name, keyPointCount, criteriaCount, keyPointCount-criteriaCount)
			if mode == CriteriaAlignmentStrict {
				errors = append(errors, msg)
			} else {
				warnings = append(warnings, msg)
			}
		}
	}

	return warnings, errors
}

