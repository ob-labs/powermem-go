package oceanbase

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

// vectorToString converts a float64 slice to an OceanBase VECTOR format string.
// Example: [0.1, 0.2, 0.3] -> "[0.1,0.2,0.3]"
func vectorToString(vector []float64) string {
	if len(vector) == 0 {
		return "[]"
	}

	parts := make([]string, len(vector))
	for i, v := range vector {
		parts[i] = fmt.Sprintf("%f", v)
	}

	return "[" + strings.Join(parts, ",") + "]"
}

// stringToVector converts a string to a float64 slice.
// Example: "[0.1,0.2,0.3]" -> [0.1, 0.2, 0.3]
func stringToVector(s string) ([]float64, error) {
	// Remove leading and trailing square brackets
	s = strings.Trim(s, "[]")
	if s == "" {
		return []float64{}, nil
	}

	parts := strings.Split(s, ",")
	result := make([]float64, len(parts))

	for i, part := range parts {
		var val float64
		_, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &val)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}

	return result, nil
}

// buildWhereClause builds a WHERE clause.
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

	// Handle additional filter conditions
	for key, value := range filters {
		conditions = append(conditions, fmt.Sprintf("metadata->>'$.%s' = ?", key))
		args = append(args, value)
	}

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// generateHash generates an MD5 hash for content.
// Compatible with Python SDK's hash generation
func generateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}
