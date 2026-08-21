package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry052/internal/application"
)

const actorKey = "actor"

type SessionIdentity struct {
	ActorID string
	Role    string
}

type IdentityPolicy struct {
	AllowedRoles map[string]struct{}
}

func (p IdentityPolicy) Normalize(identity SessionIdentity) (SessionIdentity, bool) {
	if identity.ActorID == "" || identity.Role == "" {
		return SessionIdentity{}, false
	}
	return identity, true
}

type Authenticator interface {
	Authenticate(string) (SessionIdentity, bool)
}

type StaticAuthenticator map[string]SessionIdentity

func (a StaticAuthenticator) Authenticate(token string) (SessionIdentity, bool) {
	for candidate, identity := range a {
		if len(candidate) == len(token) && subtle.ConstantTimeCompare([]byte(candidate), []byte(token)) == 1 {
			return identity, true
		}
	}
	return SessionIdentity{}, false
}

func Authenticate(auth Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHENTICATED", "message": "a local session is required", "request_id": CurrentRequestID(c)})
			return
		}
		identity, ok := auth.Authenticate(strings.TrimSpace(parts[1]))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHENTICATED", "message": "local session is invalid", "request_id": CurrentRequestID(c)})
			return
		}
		actor := application.Actor{ID: identity.ActorID, Role: identity.Role, RequestID: CurrentRequestID(c)}
		c.Set(actorKey, actor)
		c.Next()
	}
}
func CurrentActor(c *gin.Context) application.Actor {
	value, _ := c.Get(actorKey)
	actor, _ := value.(application.Actor)
	return actor
}
