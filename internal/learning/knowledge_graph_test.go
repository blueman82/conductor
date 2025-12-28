package learning

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeGraphAddNode(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		node        *KnowledgeNode
		wantErr     bool
		errContains string
	}{
		{
			name: "adds task node successfully",
			node: &KnowledgeNode{
				ID:       "task-1",
				NodeType: NodeTypeTask,
				Properties: map[string]interface{}{
					"name":        "Implement feature X",
					"task_number": "1",
				},
			},
			wantErr: false,
		},
		{
			name: "adds file node successfully",
			node: &KnowledgeNode{
				ID:       "file-main-go",
				NodeType: NodeTypeFile,
				Properties: map[string]interface{}{
					"path": "/src/main.go",
					"type": "go",
				},
			},
			wantErr: false,
		},
		{
			name: "adds agent node successfully",
			node: &KnowledgeNode{
				ID:       "agent-golang",
				NodeType: NodeTypeAgent,
				Properties: map[string]interface{}{
					"name": "golang-pro",
				},
			},
			wantErr: false,
		},
		{
			name: "adds pattern node successfully",
			node: &KnowledgeNode{
				ID:       "pattern-error-handling",
				NodeType: NodeTypePattern,
				Properties: map[string]interface{}{
					"description": "Error handling pattern",
				},
			},
			wantErr: false,
		},
		{
			name:        "returns error for nil node",
			node:        nil,
			wantErr:     true,
			errContains: "node cannot be nil",
		},
		{
			name: "auto-generates UUID when ID is empty",
			node: &KnowledgeNode{
				NodeType: NodeTypeTask,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, cleanup := setupKGTestStore(t)
			defer cleanup()

			kg := store.NewKnowledgeGraph()

			err := kg.AddNode(ctx, tt.node)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)

			// Verify node was stored
			if tt.node != nil {
				assert.NotEmpty(t, tt.node.ID)
				assert.False(t, tt.node.CreatedAt.IsZero())

				// Retrieve and verify
				retrieved, err := kg.GetNode(ctx, tt.node.ID)
				require.NoError(t, err)
				require.NotNil(t, retrieved)
				assert.Equal(t, tt.node.ID, retrieved.ID)
				assert.Equal(t, tt.node.NodeType, retrieved.NodeType)
			}
		})
	}
}

func TestKnowledgeGraphAddEdge(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		edge        *KnowledgeEdge
		wantErr     bool
		errContains string
	}{
		{
			name: "adds modifies edge successfully",
			edge: &KnowledgeEdge{
				SourceID: "task-1",
				TargetID: "file-main-go",
				EdgeType: EdgeTypeModifies,
				Weight:   0.9,
				Metadata: map[string]interface{}{
					"lines_changed": 50,
				},
			},
			wantErr: false,
		},
		{
			name: "adds succeeded_with edge successfully",
			edge: &KnowledgeEdge{
				SourceID: "task-1",
				TargetID: "agent-golang",
				EdgeType: EdgeTypeSucceededWith,
				Weight:   1.0,
			},
			wantErr: false,
		},
		{
			name: "adds similar_to edge successfully",
			edge: &KnowledgeEdge{
				SourceID: "file-main-go",
				TargetID: "file-util-go",
				EdgeType: EdgeTypeSimilarTo,
				Weight:   0.85,
			},
			wantErr: false,
		},
		{
			name: "adds caused_failure edge successfully",
			edge: &KnowledgeEdge{
				SourceID: "pattern-memory-leak",
				TargetID: "task-1",
				EdgeType: EdgeTypeCausedFailure,
				Weight:   0.7,
			},
			wantErr: false,
		},
		{
			name:        "returns error for nil edge",
			edge:        nil,
			wantErr:     true,
			errContains: "edge cannot be nil",
		},
		{
			name: "returns error for missing source_id",
			edge: &KnowledgeEdge{
				SourceID: "",
				TargetID: "file-main-go",
				EdgeType: EdgeTypeModifies,
			},
			wantErr:     true,
			errContains: "source_id and target_id are required",
		},
		{
			name: "returns error for missing target_id",
			edge: &KnowledgeEdge{
				SourceID: "task-1",
				TargetID: "",
				EdgeType: EdgeTypeModifies,
			},
			wantErr:     true,
			errContains: "source_id and target_id are required",
		},
		{
			name: "defaults weight to 1.0 if not set",
			edge: &KnowledgeEdge{
				SourceID: "task-1",
				TargetID: "file-main-go",
				EdgeType: EdgeTypeModifies,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, cleanup := setupKGTestStore(t)
			defer cleanup()

			kg := store.NewKnowledgeGraph()

			// Add nodes first
			kg.AddNode(ctx, &KnowledgeNode{ID: "task-1", NodeType: NodeTypeTask})
			kg.AddNode(ctx, &KnowledgeNode{ID: "file-main-go", NodeType: NodeTypeFile})
			kg.AddNode(ctx, &KnowledgeNode{ID: "file-util-go", NodeType: NodeTypeFile})
			kg.AddNode(ctx, &KnowledgeNode{ID: "agent-golang", NodeType: NodeTypeAgent})
			kg.AddNode(ctx, &KnowledgeNode{ID: "pattern-memory-leak", NodeType: NodeTypePattern})

			err := kg.AddEdge(ctx, tt.edge)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)

			if tt.edge != nil {
				assert.Greater(t, tt.edge.ID, int64(0))
				assert.False(t, tt.edge.CreatedAt.IsZero())

				// Verify default weight
				if tt.name == "defaults weight to 1.0 if not set" {
					assert.Equal(t, 1.0, tt.edge.Weight)
				}
			}
		})
	}
}

func TestKnowledgeGraphGetRelated(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupKGTestStore(t)
	defer cleanup()

	kg := store.NewKnowledgeGraph()

	// Build a graph:
	// task-1 --modifies--> file-a
	// task-1 --succeeded_with--> agent-go
	// file-a --similar_to--> file-b
	// file-b --modifies--> task-2 (reverse relationship)
	// task-2 --succeeded_with--> agent-py

	nodes := []*KnowledgeNode{
		{ID: "task-1", NodeType: NodeTypeTask, Properties: map[string]interface{}{"name": "Task 1"}},
		{ID: "task-2", NodeType: NodeTypeTask, Properties: map[string]interface{}{"name": "Task 2"}},
		{ID: "file-a", NodeType: NodeTypeFile, Properties: map[string]interface{}{"path": "/a.go"}},
		{ID: "file-b", NodeType: NodeTypeFile, Properties: map[string]interface{}{"path": "/b.go"}},
		{ID: "agent-go", NodeType: NodeTypeAgent, Properties: map[string]interface{}{"name": "golang-pro"}},
		{ID: "agent-py", NodeType: NodeTypeAgent, Properties: map[string]interface{}{"name": "python-pro"}},
	}

	for _, n := range nodes {
		err := kg.AddNode(ctx, n)
		require.NoError(t, err)
	}

	edges := []*KnowledgeEdge{
		{SourceID: "task-1", TargetID: "file-a", EdgeType: EdgeTypeModifies},
		{SourceID: "task-1", TargetID: "agent-go", EdgeType: EdgeTypeSucceededWith},
		{SourceID: "file-a", TargetID: "file-b", EdgeType: EdgeTypeSimilarTo},
		{SourceID: "task-2", TargetID: "file-b", EdgeType: EdgeTypeModifies},
		{SourceID: "task-2", TargetID: "agent-py", EdgeType: EdgeTypeSucceededWith},
	}

	for _, e := range edges {
		err := kg.AddEdge(ctx, e)
		require.NoError(t, err)
	}

	t.Run("finds direct neighbors (1 hop)", func(t *testing.T) {
		related, err := kg.GetRelated(ctx, "task-1", 1, nil)
		require.NoError(t, err)

		// Should find file-a and agent-go
		assert.Len(t, related, 2)

		ids := make(map[string]bool)
		for _, n := range related {
			ids[n.ID] = true
		}
		assert.True(t, ids["file-a"])
		assert.True(t, ids["agent-go"])
	})

	t.Run("finds nodes within 2 hops", func(t *testing.T) {
		related, err := kg.GetRelated(ctx, "task-1", 2, nil)
		require.NoError(t, err)

		// Should find file-a, agent-go (hop 1) + file-b (hop 2)
		assert.Len(t, related, 3)

		ids := make(map[string]bool)
		for _, n := range related {
			ids[n.ID] = true
		}
		assert.True(t, ids["file-a"])
		assert.True(t, ids["agent-go"])
		assert.True(t, ids["file-b"])
	})

	t.Run("finds nodes within 3 hops", func(t *testing.T) {
		related, err := kg.GetRelated(ctx, "task-1", 3, nil)
		require.NoError(t, err)

		// Hop 1: file-a, agent-go
		// Hop 2: file-b (via file-a)
		// Hop 3: task-2 (via file-b), agent-py would be hop 4
		// So within 3 hops: file-a, agent-go, file-b, task-2 = 4 nodes
		assert.Len(t, related, 4)
	})

	t.Run("filters by edge type", func(t *testing.T) {
		related, err := kg.GetRelated(ctx, "task-1", 2, []EdgeType{EdgeTypeModifies})
		require.NoError(t, err)

		// Only follows modifies edges: task-1 -> file-a (via modifies)
		// file-a has no outgoing modifies edges, and the similar_to edge is filtered out
		// So only file-a is found
		ids := make(map[string]bool)
		for _, n := range related {
			ids[n.ID] = true
		}
		assert.True(t, ids["file-a"])
		assert.Len(t, related, 1, "Should only find file-a when filtering by EdgeTypeModifies")
	})

	t.Run("handles node with no edges", func(t *testing.T) {
		// Add isolated node
		err := kg.AddNode(ctx, &KnowledgeNode{ID: "isolated", NodeType: NodeTypeTask})
		require.NoError(t, err)

		related, err := kg.GetRelated(ctx, "isolated", 2, nil)
		require.NoError(t, err)
		assert.Len(t, related, 0)
	})

	t.Run("respects max hop limit of 10", func(t *testing.T) {
		related, err := kg.GetRelated(ctx, "task-1", 100, nil)
		require.NoError(t, err)
		// Should still work, just capped at 10 hops
		assert.NotNil(t, related)
	})

	t.Run("defaults to 1 hop for zero or negative hops", func(t *testing.T) {
		related, err := kg.GetRelated(ctx, "task-1", 0, nil)
		require.NoError(t, err)
		// Should find direct neighbors only
		assert.Len(t, related, 2)
	})
}

func TestKnowledgeGraphFindPath(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupKGTestStore(t)
	defer cleanup()

	kg := store.NewKnowledgeGraph()

	// Build a graph:
	// A -> B -> C -> D
	//      |
	//      v
	//      E

	nodes := []*KnowledgeNode{
		{ID: "A", NodeType: NodeTypeTask},
		{ID: "B", NodeType: NodeTypeFile},
		{ID: "C", NodeType: NodeTypeFile},
		{ID: "D", NodeType: NodeTypeAgent},
		{ID: "E", NodeType: NodeTypePattern},
	}

	for _, n := range nodes {
		err := kg.AddNode(ctx, n)
		require.NoError(t, err)
	}

	edges := []*KnowledgeEdge{
		{SourceID: "A", TargetID: "B", EdgeType: EdgeTypeModifies},
		{SourceID: "B", TargetID: "C", EdgeType: EdgeTypeSimilarTo},
		{SourceID: "C", TargetID: "D", EdgeType: EdgeTypeSucceededWith},
		{SourceID: "B", TargetID: "E", EdgeType: EdgeTypeCausedFailure},
	}

	for _, e := range edges {
		err := kg.AddEdge(ctx, e)
		require.NoError(t, err)
	}

	t.Run("finds shortest path A to D", func(t *testing.T) {
		path, err := kg.FindPath(ctx, "A", "D")
		require.NoError(t, err)
		require.NotNil(t, path)

		// Path should be A -> B -> C -> D
		assert.Len(t, path, 4)
		assert.Equal(t, "A", path[0].ID)
		assert.Equal(t, "B", path[1].ID)
		assert.Equal(t, "C", path[2].ID)
		assert.Equal(t, "D", path[3].ID)
	})

	t.Run("finds shortest path A to E", func(t *testing.T) {
		path, err := kg.FindPath(ctx, "A", "E")
		require.NoError(t, err)
		require.NotNil(t, path)

		// Path should be A -> B -> E
		assert.Len(t, path, 3)
		assert.Equal(t, "A", path[0].ID)
		assert.Equal(t, "B", path[1].ID)
		assert.Equal(t, "E", path[2].ID)
	})

	t.Run("returns single node path for same source and target", func(t *testing.T) {
		path, err := kg.FindPath(ctx, "A", "A")
		require.NoError(t, err)
		require.NotNil(t, path)

		assert.Len(t, path, 1)
		assert.Equal(t, "A", path[0].ID)
	})

	t.Run("returns nil for disconnected nodes", func(t *testing.T) {
		// Add isolated node
		err := kg.AddNode(ctx, &KnowledgeNode{ID: "Z", NodeType: NodeTypeTask})
		require.NoError(t, err)

		path, err := kg.FindPath(ctx, "A", "Z")
		require.NoError(t, err)
		assert.Nil(t, path)
	})

	t.Run("returns nil for non-existent target", func(t *testing.T) {
		path, err := kg.FindPath(ctx, "A", "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, path)
	})
}

func TestKnowledgeGraphDeleteNode(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupKGTestStore(t)
	defer cleanup()

	kg := store.NewKnowledgeGraph()

	// Create nodes and edges
	err := kg.AddNode(ctx, &KnowledgeNode{ID: "task-1", NodeType: NodeTypeTask})
	require.NoError(t, err)
	err = kg.AddNode(ctx, &KnowledgeNode{ID: "file-a", NodeType: NodeTypeFile})
	require.NoError(t, err)
	err = kg.AddNode(ctx, &KnowledgeNode{ID: "agent-go", NodeType: NodeTypeAgent})
	require.NoError(t, err)

	err = kg.AddEdge(ctx, &KnowledgeEdge{SourceID: "task-1", TargetID: "file-a", EdgeType: EdgeTypeModifies})
	require.NoError(t, err)
	err = kg.AddEdge(ctx, &KnowledgeEdge{SourceID: "task-1", TargetID: "agent-go", EdgeType: EdgeTypeSucceededWith})
	require.NoError(t, err)

	t.Run("deletes node and its edges", func(t *testing.T) {
		err := kg.DeleteNode(ctx, "task-1")
		require.NoError(t, err)

		// Verify node is deleted
		node, err := kg.GetNode(ctx, "task-1")
		require.NoError(t, err)
		assert.Nil(t, node)

		// Verify edges are deleted
		edges, err := kg.GetEdges(ctx, "file-a", nil)
		require.NoError(t, err)
		assert.Len(t, edges, 0)
	})

	t.Run("handles deleting non-existent node gracefully", func(t *testing.T) {
		err := kg.DeleteNode(ctx, "nonexistent")
		require.NoError(t, err) // Should not error
	})
}

func TestKnowledgeGraphGetEdges(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupKGTestStore(t)
	defer cleanup()

	kg := store.NewKnowledgeGraph()

	// Create nodes
	kg.AddNode(ctx, &KnowledgeNode{ID: "task-1", NodeType: NodeTypeTask})
	kg.AddNode(ctx, &KnowledgeNode{ID: "file-a", NodeType: NodeTypeFile})
	kg.AddNode(ctx, &KnowledgeNode{ID: "agent-go", NodeType: NodeTypeAgent})

	// Create edges
	kg.AddEdge(ctx, &KnowledgeEdge{SourceID: "task-1", TargetID: "file-a", EdgeType: EdgeTypeModifies, Weight: 0.8})
	kg.AddEdge(ctx, &KnowledgeEdge{SourceID: "task-1", TargetID: "agent-go", EdgeType: EdgeTypeSucceededWith, Weight: 1.0})

	t.Run("gets all edges for a node", func(t *testing.T) {
		edges, err := kg.GetEdges(ctx, "task-1", nil)
		require.NoError(t, err)
		assert.Len(t, edges, 2)
	})

	t.Run("filters edges by type", func(t *testing.T) {
		// Get only modifies edges for task-1
		edges, err := kg.GetEdges(ctx, "task-1", []EdgeType{EdgeTypeModifies})
		require.NoError(t, err)
		require.Len(t, edges, 1, "Should have 1 modifies edge")
		assert.Equal(t, EdgeTypeModifies, edges[0].EdgeType)
		assert.Equal(t, "task-1", edges[0].SourceID)
		assert.Equal(t, "file-a", edges[0].TargetID)
	})

	t.Run("returns edges where node is target", func(t *testing.T) {
		edges, err := kg.GetEdges(ctx, "file-a", nil)
		require.NoError(t, err)
		assert.Len(t, edges, 1)
		assert.Equal(t, "task-1", edges[0].SourceID)
		assert.Equal(t, "file-a", edges[0].TargetID)
	})

	t.Run("returns empty for node with no edges", func(t *testing.T) {
		kg.AddNode(ctx, &KnowledgeNode{ID: "isolated", NodeType: NodeTypeTask})
		edges, err := kg.GetEdges(ctx, "isolated", nil)
		require.NoError(t, err)
		assert.Len(t, edges, 0)
	})
}

func TestKnowledgeGraphMigration(t *testing.T) {
	t.Run("kg_nodes table is created after migration", func(t *testing.T) {
		store, cleanup := setupKGTestStore(t)
		defer cleanup()

		exists, err := store.tableExists("kg_nodes")
		require.NoError(t, err)
		assert.True(t, exists, "kg_nodes table should exist after migration")
	})

	t.Run("kg_edges table is created after migration", func(t *testing.T) {
		store, cleanup := setupKGTestStore(t)
		defer cleanup()

		exists, err := store.tableExists("kg_edges")
		require.NoError(t, err)
		assert.True(t, exists, "kg_edges table should exist after migration")
	})

	t.Run("kg_edges indexes are created", func(t *testing.T) {
		store, cleanup := setupKGTestStore(t)
		defer cleanup()

		indexes := []string{
			"idx_kg_edges_source",
			"idx_kg_edges_target",
			"idx_kg_edges_type",
		}

		for _, idx := range indexes {
			exists, err := store.indexExists(idx)
			require.NoError(t, err)
			assert.True(t, exists, "index %s should exist", idx)
		}
	})
}

func TestStoreConvenienceMethods(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupKGTestStore(t)
	defer cleanup()

	kg := store.NewKnowledgeGraph()

	// Create test nodes
	kg.AddNode(ctx, &KnowledgeNode{ID: "task-1", NodeType: NodeTypeTask})
	kg.AddNode(ctx, &KnowledgeNode{ID: "file-a", NodeType: NodeTypeFile})
	kg.AddNode(ctx, &KnowledgeNode{ID: "agent-go", NodeType: NodeTypeAgent})
	kg.AddNode(ctx, &KnowledgeNode{ID: "pattern-1", NodeType: NodeTypePattern})

	t.Run("RecordTaskFileRelation creates edge", func(t *testing.T) {
		err := store.RecordTaskFileRelation(ctx, "task-1", "file-a", 0.9, map[string]interface{}{
			"lines_changed": 50,
		})
		require.NoError(t, err)

		edges, err := kg.GetEdges(ctx, "task-1", []EdgeType{EdgeTypeModifies})
		require.NoError(t, err)
		assert.Len(t, edges, 1)
		assert.Equal(t, 0.9, edges[0].Weight)
	})

	t.Run("RecordTaskAgentSuccess creates edge", func(t *testing.T) {
		err := store.RecordTaskAgentSuccess(ctx, "task-1", "agent-go", 1.0, nil)
		require.NoError(t, err)

		edges, err := kg.GetEdges(ctx, "task-1", []EdgeType{EdgeTypeSucceededWith})
		require.NoError(t, err)
		assert.Len(t, edges, 1)
	})

	t.Run("RecordFileSimilarity creates edge", func(t *testing.T) {
		kg.AddNode(ctx, &KnowledgeNode{ID: "file-b", NodeType: NodeTypeFile})

		err := store.RecordFileSimilarity(ctx, "file-a", "file-b", 0.85, nil)
		require.NoError(t, err)

		edges, err := kg.GetEdges(ctx, "file-a", []EdgeType{EdgeTypeSimilarTo})
		require.NoError(t, err)
		assert.Len(t, edges, 1)
		assert.Equal(t, 0.85, edges[0].Weight)
	})

	t.Run("RecordPatternFailure creates edge", func(t *testing.T) {
		err := store.RecordPatternFailure(ctx, "pattern-1", "task-1", 0.7, nil)
		require.NoError(t, err)

		edges, err := kg.GetEdges(ctx, "pattern-1", []EdgeType{EdgeTypeCausedFailure})
		require.NoError(t, err)
		assert.Len(t, edges, 1)
	})

	t.Run("FindAgentsForFile finds related agents", func(t *testing.T) {
		// task-1 modifies file-a and succeeded_with agent-go
		// So file-a is connected to agent-go via task-1

		agents, err := store.FindAgentsForFile(ctx, "file-a", 2)
		require.NoError(t, err)
		assert.Len(t, agents, 1)
		assert.Equal(t, "agent-go", agents[0].ID)
	})
}

func TestKnowledgeGraphInterface(t *testing.T) {
	// Verify SQLiteKnowledgeGraph implements KnowledgeGraph
	var _ KnowledgeGraph = (*SQLiteKnowledgeGraph)(nil)
}

// setupKGTestStore creates a test store with a temporary database for knowledge graph tests
func setupKGTestStore(t *testing.T) (*Store, func()) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test_kg.db")
	store, err := NewStore(dbPath)
	require.NoError(t, err)

	cleanup := func() {
		store.Close()
	}

	return store, cleanup
}
