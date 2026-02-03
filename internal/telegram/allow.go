package telegram

import (
	"strconv"
	"strings"
)

func isAllowedUser(allow []string, user *User) bool {
	if user == nil {
		return false
	}
	fromID := strconv.FormatInt(user.ID, 10)
	username := strings.ToLower(strings.TrimPrefix(user.Username, "@"))
	for _, allowed := range allow {
		val := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(allowed), "@"))
		if val == "" {
			continue
		}
		if val == fromID || val == username {
			return true
		}
	}
	return false
}

func isAllowedUpdate(allow []string, update Update) bool {
	if update.Message == nil {
		return false
	}
	return isAllowedUser(allow, update.Message.From)
}
