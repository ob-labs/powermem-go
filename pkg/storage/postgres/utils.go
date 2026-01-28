package postgres

import (
	"fmt"
	"strings"
)

// buildWhereClause builds a WHERE clause starting from $1.
func buildWhereClause(userID, agentID string, filters map[string]interface{}) (string, []interface{}) {
	return buildWhereClauseWithOffset(userID, agentID, filters, 1)
}

// buildWhereClauseWithOffset builds a WHERE clause starting from a specific parameter index.
func buildWhereClauseWithOffset(userID, agentID string, filters map[string]interface{}, startIndex int) (string, []interface{}) {
	conditions := []string{}
	args := []interface{}{}
	argIndex := startIndex

	if userID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, userID)
		argIndex++
	}

	if agentID != "" {
		conditions = append(conditions, fmt.Sprintf("agent_id = $%d", argIndex))
		args = append(args, agentID)
		// argIndex++ // 为未来扩展预留
	}

	// Note: Currently not processing filters map for metadata conditions
	// This would require JSON operations in PostgreSQL

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}
