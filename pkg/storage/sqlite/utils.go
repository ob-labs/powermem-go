package sqlite

import (
	"strings"
)

// buildWhereClause builds a WHERE clause (fixed version).
func buildWhereClause(userID, agentID string, filters map[string]interface{}) (string, []interface{}) {
	conditions := []string{}
	args := []interface{}{}

	if userID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, userID)
	}

	if agentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, agentID)
	}

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}
