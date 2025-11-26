package behavioral

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	projectsDir := filepath.Join(tmpDir, ".claude", "projects")

	t.Run("no projects directory", func(t *testing.T) {
		projects, err := ListProjects()
		require.NoError(t, err)
		assert.Empty(t, projects)
	})

	t.Run("empty projects directory", func(t *testing.T) {
		err := os.MkdirAll(projectsDir, 0755)
		require.NoError(t, err)

		projects, err := ListProjects()
		require.NoError(t, err)
		assert.Empty(t, projects)
	})

	t.Run("single project with sessions", func(t *testing.T) {
		project1 := filepath.Join(projectsDir, "test-project-1")
		err := os.MkdirAll(project1, 0755)
		require.NoError(t, err)

		// Create test JSONL files
		session1 := filepath.Join(project1, "session1.jsonl")
		session2 := filepath.Join(project1, "session2.jsonl")
		err = os.WriteFile(session1, []byte("test data 1\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(session2, []byte("test data 2 longer\n"), 0644)
		require.NoError(t, err)

		projects, err := ListProjects()
		require.NoError(t, err)
		assert.Len(t, projects, 1)
		assert.Equal(t, "test-project-1", projects[0].Name)
		assert.Equal(t, 2, projects[0].SessionCount)
		assert.Greater(t, projects[0].TotalSize, int64(0))
	})

	t.Run("multiple projects sorted by name", func(t *testing.T) {
		project2 := filepath.Join(projectsDir, "alpha-project")
		project3 := filepath.Join(projectsDir, "zulu-project")
		err := os.MkdirAll(project2, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(project3, 0755)
		require.NoError(t, err)

		// Add sessions to alpha
		session := filepath.Join(project2, "session.jsonl")
		err = os.WriteFile(session, []byte("data\n"), 0644)
		require.NoError(t, err)

		projects, err := ListProjects()
		require.NoError(t, err)
		assert.Len(t, projects, 3)
		assert.Equal(t, "alpha-project", projects[0].Name)
		assert.Equal(t, "test-project-1", projects[1].Name)
		assert.Equal(t, "zulu-project", projects[2].Name)
	})

	t.Run("ignores non-directory files", func(t *testing.T) {
		file := filepath.Join(projectsDir, "not-a-project.txt")
		err := os.WriteFile(file, []byte("ignore me\n"), 0644)
		require.NoError(t, err)

		projects, err := ListProjects()
		require.NoError(t, err)
		assert.Len(t, projects, 3) // Still 3 directories
	})

	t.Run("handles project with no sessions", func(t *testing.T) {
		emptyProject := filepath.Join(projectsDir, "empty-project")
		err := os.MkdirAll(emptyProject, 0755)
		require.NoError(t, err)

		projects, err := ListProjects()
		require.NoError(t, err)

		var found bool
		for _, p := range projects {
			if p.Name == "empty-project" {
				found = true
				assert.Equal(t, 0, p.SessionCount)
				assert.Equal(t, int64(0), p.TotalSize)
			}
		}
		assert.True(t, found)
	})
}

func TestGetProjectStats(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	projectsDir := filepath.Join(tmpDir, ".claude", "projects")
	err := os.MkdirAll(projectsDir, 0755)
	require.NoError(t, err)

	t.Run("project not found", func(t *testing.T) {
		_, err := GetProjectStats("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project not found")
	})

	t.Run("valid project with sessions", func(t *testing.T) {
		projectName := "test-project"
		projectPath := filepath.Join(projectsDir, projectName)
		err := os.MkdirAll(projectPath, 0755)
		require.NoError(t, err)

		// Create test sessions
		session1 := filepath.Join(projectPath, "session1.jsonl")
		session2 := filepath.Join(projectPath, "session2.jsonl")
		err = os.WriteFile(session1, []byte("test data 1\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(session2, []byte("test data 2\n"), 0644)
		require.NoError(t, err)

		stats, err := GetProjectStats(projectName)
		require.NoError(t, err)
		assert.Equal(t, 2, stats.TotalSessions)
		assert.Greater(t, stats.TotalSize, int64(0))
		assert.NotEmpty(t, stats.LastModified)
	})

	t.Run("project with no sessions", func(t *testing.T) {
		projectName := "empty-project"
		projectPath := filepath.Join(projectsDir, projectName)
		err := os.MkdirAll(projectPath, 0755)
		require.NoError(t, err)

		stats, err := GetProjectStats(projectName)
		require.NoError(t, err)
		assert.Equal(t, 0, stats.TotalSessions)
		assert.Equal(t, int64(0), stats.TotalSize)
		assert.Empty(t, stats.LastModified)
	})
}

func TestProjectInfo(t *testing.T) {
	t.Run("basic fields", func(t *testing.T) {
		info := ProjectInfo{
			Name:         "test-project",
			Path:         "/path/to/project",
			SessionCount: 5,
			TotalSize:    1024,
		}

		assert.Equal(t, "test-project", info.Name)
		assert.Equal(t, "/path/to/project", info.Path)
		assert.Equal(t, 5, info.SessionCount)
		assert.Equal(t, int64(1024), info.TotalSize)
	})
}

func TestProjectStats(t *testing.T) {
	t.Run("basic fields", func(t *testing.T) {
		stats := ProjectStats{
			TotalSessions: 10,
			SuccessRate:   0.85,
			TotalSize:     2048,
			LastModified:  "2025-01-01 12:00:00",
			ErrorRate:     0.15,
		}

		assert.Equal(t, 10, stats.TotalSessions)
		assert.Equal(t, 0.85, stats.SuccessRate)
		assert.Equal(t, int64(2048), stats.TotalSize)
		assert.Equal(t, "2025-01-01 12:00:00", stats.LastModified)
		assert.Equal(t, 0.15, stats.ErrorRate)
	})
}
