package auth

import "context"

const userIDKey string = "userID"

func UserIDFromContext(ctx context.Context) (int64, bool) {
	value := ctx.Value(userIDKey)
	userID, ok := value.(int64)
	return userID, ok
}
