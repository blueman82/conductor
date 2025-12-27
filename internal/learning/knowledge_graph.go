package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NodeType represents the type of a knowledge graph node
type NodeType string

const (
	// NodeTypeTask represents a task node
	NodeTypeTask NodeType = "task"
	// NodeTypeFile represents a file node
	NodeTypeFile NodeType = "file"
	// NodeTypeAgent represents an agent node
	NodeTypeAgent NodeType = "agent"
	// NodeTypePattern represents a pattern node
	NodeTypePattern NodeType = "pattern"
)

// EdgeType represents the type of relationship between nodes
type EdgeType string

const (
	// EdgeTypeModifies represents "Task X modifies File Y"
	EdgeTypeModifies EdgeType = "modifies"
	// EdgeTypeSucceededWith represents "Task X succeeded with Agent Z"
	EdgeTypeSucceededWith EdgeType = "succeeded_with"
	// EdgeTypeSimilarTo represents "File Y is similar to File W"
	EdgeTypeSimilarTo EdgeType = "similar_to"
	// EdgeTypeCausedFailure represents "Pattern P caused failure in Task X"
	EdgeTypeCausedFailure EdgeType = "caused_failure"
	// EdgeTypeDependsOn represents "Task X depends on Task Y"
	EdgeTypeDependsOn EdgeType = "depends_on"
	// EdgeTypeUsedBy represents "Agent X was used by Task Y"
	EdgeTypeUsedBy EdgeType = "used_by"
)

// KnowledgeNode represents a node in the knowledge graph
type KnowledgeNode struct {
	// ID is a unique identifier (UUID) for the node
	ID string `json:"id"`

	// NodeType is the type of entity this node represents
	NodeType NodeType `json:"node_type"`

	// Properties stores arbitrary key-value attributes for the node
	Properties map[string]interface{} `json:"properties,omitempty"`

	// CreatedAt is when the node was created
	CreatedAt time.Time `json:"created_at"`
}

// KnowledgeEdge represents a directed relationship between two nodes
type KnowledgeEdge struct {
	// ID is a unique identifier for the edge
	ID int64 `json:"id,omitempty"`

	// SourceID is the ID of the source node
	SourceID string `json:"source_id"`

	// TargetID is the ID of the target node
	TargetID string `json:"target_id"`

	// EdgeType is the type of relationship
	EdgeType EdgeType `json:"edge_type"`

	// Weight is an optional edge weight (0.0-1.0) for scoring relevance
	Weight float64 `json:"weight"`

	// Metadata stores additional edge-specific information as JSON
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt is when the edge was created
	CreatedAt time.Time `json:"created_at"`
}

// KnowledgeGraph defines the interface for graph operations
type KnowledgeGraph interface {
	// AddNode adds a new node to the graph
	AddNode(ctx context.Context, node *KnowledgeNode) error

	// AddEdge adds a new edge between two nodes
	AddEdge(ctx context.Context, edge *KnowledgeEdge) error

	// GetRelated returns nodes related to the given node within N hops
	// optionally filtered by edge types
	GetRelated(ctx context.Context, nodeID string, hops int, edgeTypes []EdgeType) ([]KnowledgeNode, error)

	// FindPath finds the shortest path between two nodes using BFS
	// Returns nil if no path exists
	FindPath(ctx context.Context, fromID, toID string) ([]KnowledgeNode, error)

	// GetNode retrieves a node by ID
	GetNode(ctx context.Context, nodeID string) (*KnowledgeNode, error)

	// GetEdges retrieves edges for a node, optionally filtered by type
	GetEdges(ctx context.Context, nodeID string, edgeTypes []EdgeType) ([]KnowledgeEdge, error)

	// DeleteNode removes a node and all its edges
	DeleteNode(ctx context.Context, nodeID string) error
}

// SQLiteKnowledgeGraph implements KnowledgeGraph using SQLite
type SQLiteKnowledgeGraph struct {
	db *sql.DB
}

// NewKnowledgeGraph creates a new SQLiteKnowledgeGraph using the Store's database
func (s *Store) NewKnowledgeGraph() *SQLiteKnowledgeGraph {
	return &SQLiteKnowledgeGraph{db: s.db}
}

// AddNode adds a new node to the knowledge graph
func (kg *SQLiteKnowledgeGraph) AddNode(ctx context.Context, node *KnowledgeNode) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	// Generate UUID if not provided
	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	// Set creation time
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}

	// Marshal properties to JSON
	propertiesJSON := "{}"
	if node.Properties != nil {
		data, err := json.Marshal(node.Properties)
		if err != nil {
			return fmt.Errorf("marshal properties: %w", err)
		}
		propertiesJSON = string(data)
	}

	query := `INSERT INTO kg_nodes (id, node_type, properties, created_at)
		VALUES (?, ?, ?, ?)`

	_, err := kg.db.ExecContext(ctx, query,
		node.ID,
		string(node.NodeType),
		propertiesJSON,
		node.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert node: %w", err)
	}

	return nil
}

// AddEdge adds a new edge between two nodes
func (kg *SQLiteKnowledgeGraph) AddEdge(ctx context.Context, edge *KnowledgeEdge) error {
	if edge == nil {
		return fmt.Errorf("edge cannot be nil")
	}

	if edge.SourceID == "" || edge.TargetID == "" {
		return fmt.Errorf("source_id and target_id are required")
	}

	// Default weight to 1.0 if not set
	if edge.Weight == 0 {
		edge.Weight = 1.0
	}

	// Set creation time
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now()
	}

	// Marshal metadata to JSON
	metadataJSON := "{}"
	if edge.Metadata != nil {
		data, err := json.Marshal(edge.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		metadataJSON = string(data)
	}

	query := `INSERT INTO kg_edges (source_id, target_id, edge_type, weight, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	result, err := kg.db.ExecContext(ctx, query,
		edge.SourceID,
		edge.TargetID,
		string(edge.EdgeType),
		edge.Weight,
		metadataJSON,
		edge.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert edge: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	edge.ID = id

	return nil
}

// GetNode retrieves a node by ID
func (kg *SQLiteKnowledgeGraph) GetNode(ctx context.Context, nodeID string) (*KnowledgeNode, error) {
	query := `SELECT id, node_type, properties, created_at FROM kg_nodes WHERE id = ?`

	row := kg.db.QueryRowContext(ctx, query, nodeID)

	node := &KnowledgeNode{}
	var nodeType, propertiesJSON string

	err := row.Scan(&node.ID, &nodeType, &propertiesJSON, &node.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan node: %w", err)
	}

	node.NodeType = NodeType(nodeType)
	if propertiesJSON != "" && propertiesJSON != "{}" {
		if err := json.Unmarshal([]byte(propertiesJSON), &node.Properties); err != nil {
			return nil, fmt.Errorf("unmarshal properties: %w", err)
		}
	}

	return node, nil
}

// GetEdges retrieves edges for a node, optionally filtered by type
func (kg *SQLiteKnowledgeGraph) GetEdges(ctx context.Context, nodeID string, edgeTypes []EdgeType) ([]KnowledgeEdge, error) {
	query := `SELECT id, source_id, target_id, edge_type, weight, metadata, created_at
		FROM kg_edges WHERE (source_id = ? OR target_id = ?)`
	args := []interface{}{nodeID, nodeID}

	if len(edgeTypes) > 0 {
		placeholders := make([]string, len(edgeTypes))
		for i, et := range edgeTypes {
			placeholders[i] = "?"
			args = append(args, string(et))
		}
		query += fmt.Sprintf(" AND edge_type IN (%s)", joinStrings(placeholders, ","))
	}

	rows, err := kg.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer rows.Close()

	var edges []KnowledgeEdge
	for rows.Next() {
		var edge KnowledgeEdge
		var edgeType, metadataJSON string

		err := rows.Scan(
			&edge.ID,
			&edge.SourceID,
			&edge.TargetID,
			&edgeType,
			&edge.Weight,
			&metadataJSON,
			&edge.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}

		edge.EdgeType = EdgeType(edgeType)
		if metadataJSON != "" && metadataJSON != "{}" {
			if err := json.Unmarshal([]byte(metadataJSON), &edge.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		edges = append(edges, edge)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate edges: %w", err)
	}

	return edges, nil
}

// GetRelated returns nodes related to the given node within N hops using BFS
func (kg *SQLiteKnowledgeGraph) GetRelated(ctx context.Context, nodeID string, hops int, edgeTypes []EdgeType) ([]KnowledgeNode, error) {
	if hops <= 0 {
		hops = 1
	}
	if hops > 10 {
		hops = 10 // Safety limit
	}

	// BFS to find related nodes
	visited := make(map[string]bool)
	visited[nodeID] = true

	queue := []string{nodeID}
	currentHop := 0

	var relatedNodeIDs []string

	for len(queue) > 0 && currentHop < hops {
		// Process all nodes at current level
		levelSize := len(queue)
		for i := 0; i < levelSize; i++ {
			currentID := queue[0]
			queue = queue[1:]

			// Get neighbors
			neighbors, err := kg.getNeighborIDs(ctx, currentID, edgeTypes)
			if err != nil {
				return nil, fmt.Errorf("get neighbors: %w", err)
			}

			for _, neighborID := range neighbors {
				if !visited[neighborID] {
					visited[neighborID] = true
					queue = append(queue, neighborID)
					relatedNodeIDs = append(relatedNodeIDs, neighborID)
				}
			}
		}
		currentHop++
	}

	// Fetch full node data for all related nodes
	var nodes []KnowledgeNode
	for _, id := range relatedNodeIDs {
		node, err := kg.GetNode(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get node %s: %w", id, err)
		}
		if node != nil {
			nodes = append(nodes, *node)
		}
	}

	return nodes, nil
}

// getNeighborIDs returns the IDs of nodes connected to the given node
func (kg *SQLiteKnowledgeGraph) getNeighborIDs(ctx context.Context, nodeID string, edgeTypes []EdgeType) ([]string, error) {
	// Query for both outgoing and incoming edges
	query := `SELECT DISTINCT
		CASE WHEN source_id = ? THEN target_id ELSE source_id END as neighbor_id
		FROM kg_edges WHERE (source_id = ? OR target_id = ?)`
	args := []interface{}{nodeID, nodeID, nodeID}

	if len(edgeTypes) > 0 {
		placeholders := make([]string, len(edgeTypes))
		for i, et := range edgeTypes {
			placeholders[i] = "?"
			args = append(args, string(et))
		}
		query += fmt.Sprintf(" AND edge_type IN (%s)", joinStrings(placeholders, ","))
	}

	rows, err := kg.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query neighbors: %w", err)
	}
	defer rows.Close()

	var neighbors []string
	for rows.Next() {
		var neighborID string
		if err := rows.Scan(&neighborID); err != nil {
			return nil, fmt.Errorf("scan neighbor: %w", err)
		}
		neighbors = append(neighbors, neighborID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate neighbors: %w", err)
	}

	return neighbors, nil
}

// FindPath finds the shortest path between two nodes using BFS
func (kg *SQLiteKnowledgeGraph) FindPath(ctx context.Context, fromID, toID string) ([]KnowledgeNode, error) {
	if fromID == toID {
		node, err := kg.GetNode(ctx, fromID)
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, nil
		}
		return []KnowledgeNode{*node}, nil
	}

	// BFS to find shortest path
	visited := make(map[string]bool)
	parent := make(map[string]string) // Maps each node to its parent in the BFS tree

	visited[fromID] = true
	queue := []string{fromID}

	found := false
	for len(queue) > 0 && !found {
		currentID := queue[0]
		queue = queue[1:]

		neighbors, err := kg.getNeighborIDs(ctx, currentID, nil)
		if err != nil {
			return nil, fmt.Errorf("get neighbors: %w", err)
		}

		for _, neighborID := range neighbors {
			if !visited[neighborID] {
				visited[neighborID] = true
				parent[neighborID] = currentID
				queue = append(queue, neighborID)

				if neighborID == toID {
					found = true
					break
				}
			}
		}
	}

	if !found {
		return nil, nil // No path exists
	}

	// Reconstruct path from toID to fromID
	var pathIDs []string
	current := toID
	for current != "" {
		pathIDs = append([]string{current}, pathIDs...)
		current = parent[current]
	}

	// Fetch full node data
	var path []KnowledgeNode
	for _, id := range pathIDs {
		node, err := kg.GetNode(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get node %s: %w", id, err)
		}
		if node != nil {
			path = append(path, *node)
		}
	}

	return path, nil
}

// DeleteNode removes a node and all its edges
func (kg *SQLiteKnowledgeGraph) DeleteNode(ctx context.Context, nodeID string) error {
	// Delete edges first (both directions)
	edgeQuery := `DELETE FROM kg_edges WHERE source_id = ? OR target_id = ?`
	_, err := kg.db.ExecContext(ctx, edgeQuery, nodeID, nodeID)
	if err != nil {
		return fmt.Errorf("delete edges: %w", err)
	}

	// Delete node
	nodeQuery := `DELETE FROM kg_nodes WHERE id = ?`
	_, err = kg.db.ExecContext(ctx, nodeQuery, nodeID)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}

	return nil
}

// Convenience methods for the Store

// GetKnowledgeGraph returns a KnowledgeGraph instance backed by the Store's database
func (s *Store) GetKnowledgeGraph() KnowledgeGraph {
	return s.NewKnowledgeGraph()
}

// RecordTaskFileRelation creates an edge between a task and file it modifies
func (s *Store) RecordTaskFileRelation(ctx context.Context, taskID, fileID string, weight float64, metadata map[string]interface{}) error {
	kg := s.NewKnowledgeGraph()
	return kg.AddEdge(ctx, &KnowledgeEdge{
		SourceID: taskID,
		TargetID: fileID,
		EdgeType: EdgeTypeModifies,
		Weight:   weight,
		Metadata: metadata,
	})
}

// RecordTaskAgentSuccess creates an edge between a task and agent that succeeded
func (s *Store) RecordTaskAgentSuccess(ctx context.Context, taskID, agentID string, weight float64, metadata map[string]interface{}) error {
	kg := s.NewKnowledgeGraph()
	return kg.AddEdge(ctx, &KnowledgeEdge{
		SourceID: taskID,
		TargetID: agentID,
		EdgeType: EdgeTypeSucceededWith,
		Weight:   weight,
		Metadata: metadata,
	})
}

// RecordFileSimilarity creates an edge between similar files
func (s *Store) RecordFileSimilarity(ctx context.Context, fileID1, fileID2 string, similarity float64, metadata map[string]interface{}) error {
	kg := s.NewKnowledgeGraph()
	return kg.AddEdge(ctx, &KnowledgeEdge{
		SourceID: fileID1,
		TargetID: fileID2,
		EdgeType: EdgeTypeSimilarTo,
		Weight:   similarity,
		Metadata: metadata,
	})
}

// RecordPatternFailure creates an edge between a pattern and task it caused to fail
func (s *Store) RecordPatternFailure(ctx context.Context, patternID, taskID string, weight float64, metadata map[string]interface{}) error {
	kg := s.NewKnowledgeGraph()
	return kg.AddEdge(ctx, &KnowledgeEdge{
		SourceID: patternID,
		TargetID: taskID,
		EdgeType: EdgeTypeCausedFailure,
		Weight:   weight,
		Metadata: metadata,
	})
}

// FindAgentsForFile finds agents that have worked well with similar files
func (s *Store) FindAgentsForFile(ctx context.Context, fileID string, maxHops int) ([]KnowledgeNode, error) {
	kg := s.NewKnowledgeGraph()

	// Find related nodes, filtering for agent nodes
	related, err := kg.GetRelated(ctx, fileID, maxHops, []EdgeType{EdgeTypeModifies, EdgeTypeSucceededWith, EdgeTypeSimilarTo})
	if err != nil {
		return nil, err
	}

	// Filter to only agent nodes
	var agents []KnowledgeNode
	for _, node := range related {
		if node.NodeType == NodeTypeAgent {
			agents = append(agents, node)
		}
	}

	return agents, nil
}
