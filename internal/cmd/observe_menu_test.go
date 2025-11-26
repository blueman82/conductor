package cmd

import (
	"fmt"
	"testing"

	"github.com/harrison/conductor/internal/behavioral"
	"github.com/stretchr/testify/assert"
)

// MockMenuReader for testing
type MockMenuReader struct {
	inputs []string
	index  int
}

func (m *MockMenuReader) ReadString(delim byte) (string, error) {
	if m.index >= len(m.inputs) {
		return "", fmt.Errorf("EOF")
	}
	result := m.inputs[m.index] + "\n"
	m.index++
	return result, nil
}

func TestFormatMenuLine(t *testing.T) {
	project := behavioral.ProjectInfo{
		Name:         "test-project",
		SessionCount: 10,
		TotalSize:    5242880, // 5 MB
	}

	line := formatMenuLine(project, 1)
	assert.Contains(t, line, "test-project")
	assert.Contains(t, line, "10 sessions")
	assert.Contains(t, line, "5.00 MB")
}

func TestReadSelection(t *testing.T) {
	projects := []behavioral.ProjectInfo{
		{Name: "project1"},
		{Name: "project2"},
		{Name: "project3"},
	}

	t.Run("valid selection", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"2"}}
		selected, err := readSelection(reader, 3, projects)
		assert.NoError(t, err)
		assert.Equal(t, "project2", selected)
	})

	t.Run("invalid selection out of range", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"5"}}
		_, err := readSelection(reader, 3, projects)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid selection")
	})

	t.Run("quit selection", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"q"}}
		_, err := readSelection(reader, 3, projects)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("non-numeric selection", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"abc"}}
		_, err := readSelection(reader, 3, projects)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid selection")
	})
}

func TestDisplaySinglePage(t *testing.T) {
	projects := []behavioral.ProjectInfo{
		{Name: "project1", SessionCount: 5, TotalSize: 1024000},
		{Name: "project2", SessionCount: 10, TotalSize: 2048000},
	}

	t.Run("successful selection", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"1"}}
		selected, err := displaySinglePage(projects, reader)
		assert.NoError(t, err)
		assert.Equal(t, "project1", selected)
	})

	t.Run("quit selection", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"q"}}
		_, err := displaySinglePage(projects, reader)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})
}

func TestDisplayPaginated(t *testing.T) {
	// Create 20 projects for pagination testing
	projects := make([]behavioral.ProjectInfo, 20)
	for i := 0; i < 20; i++ {
		projects[i] = behavioral.ProjectInfo{
			Name:         fmt.Sprintf("project-%02d", i+1),
			SessionCount: i + 1,
			TotalSize:    int64((i + 1) * 1024000),
		}
	}

	t.Run("select from first page", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"5"}}
		selected, err := displayPaginated(projects, reader)
		assert.NoError(t, err)
		assert.Equal(t, "project-05", selected)
	})

	t.Run("navigate to next page", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"n", "16"}}
		selected, err := displayPaginated(projects, reader)
		assert.NoError(t, err)
		assert.Equal(t, "project-16", selected)
	})

	t.Run("navigate to previous page", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"n", "p", "5"}}
		selected, err := displayPaginated(projects, reader)
		assert.NoError(t, err)
		assert.Equal(t, "project-05", selected)
	})

	t.Run("quit from menu", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"q"}}
		_, err := displayPaginated(projects, reader)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("invalid input retry", func(t *testing.T) {
		reader := &MockMenuReader{inputs: []string{"invalid", "", "5"}}
		selected, err := displayPaginated(projects, reader)
		assert.NoError(t, err)
		assert.Equal(t, "project-05", selected)
	})
}
