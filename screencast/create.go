package screencast

import (
	"context"

	"github.com/godbus/dbus/v5"

	"github.com/rymdport/portal/internal/apis"
	"github.com/rymdport/portal/internal/convert"
	"github.com/rymdport/portal/internal/session"
)

const (
	createSessionCallName = interfaceName + ".CreateSession"
	sessionObjectPath     = apis.ObjectPath + "/session/"
)

// CreateSession creates a ScreenCast session.
// A successfully created session can at any time be closed,
// or may at any time be closed by the portal implementation.
func CreateSession() (*Session, error) {
	return CreateSessionContext(context.Background())
}

// CreateSessionContext is CreateSession with a context
func CreateSessionContext(ctx context.Context) (*Session, error) {
	results, err := getResponse(ctx, "", createSessionCallName, func(token string) []any {
		data := map[string]dbus.Variant{
			"handle_token":         convert.FromString(token),
			"session_handle_token": session.GenerateToken(),
		}
		return []any{data}
	})
	if err != nil {
		return nil, err
	}

	sessionHandle := dbus.ObjectPath(results["session_handle"].Value().(string))

	return &Session{path: sessionHandle}, nil
}
