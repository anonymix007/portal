package screencast

import (
	"context"
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"

	"github.com/rymdport/portal/internal/apis"
	"github.com/rymdport/portal/internal/convert"
	"github.com/rymdport/portal/internal/session"
)

const (
	selectSourcesCallName      = interfaceName + ".SelectSources"
	startCallName              = interfaceName + ".Start"
	openPipeWireRemoteCallName = interfaceName + ".OpenPipeWireRemote"
)

// Session is the value for a created ScreenCast session.
// The zero value is not usable.
type Session struct {
	path dbus.ObjectPath
	done chan struct{} // closed by Close to stop listeners
}

// Close closes the current session.
func (s *Session) Close() error {
	return session.Close(s.path)
}

// SetOnClosed sets a callback to run when the portal closes the session.
// The goroutine exits on Close even if the signal never arrives.
func (s *Session) SetOnClosed(callback func(error)) {
	done := s.ensureDone()
	go func() {
		_, err := session.OnSignalClosed(s.path, done)
		select {
		case <-done:
			return // cancelled by Close
		default:
		}
		callback(err)
	}()
}

// SelectSources configures what a screen cast session should record
func (s *Session) SelectSources(sourceTypes SourceType, multiple bool, cursorMode CursorMode, restoreToken string, persistMode PersistMode) error {
	return s.SelectSourcesContext(context.Background(), sourceTypes, multiple, cursorMode, restoreToken, persistMode)
}

func (s *Session) SelectSourcesContext(ctx context.Context, sourceTypes SourceType, multiple bool, cursorMode CursorMode, restoreToken string, persistMode PersistMode) error {
	_, err := getResponse(ctx, "", selectSourcesCallName, func(token string) []any {
		data := map[string]dbus.Variant{
			"handle_token": convert.FromString(token),
			"types":        dbus.MakeVariant(sourceTypes),
			"multiple":     dbus.MakeVariant(multiple),
			"cursor_mode":  dbus.MakeVariant(cursorMode),
			"persist_mode": dbus.MakeVariant(persistMode),
		}
		if restoreToken != "" {
			data["restore_token"] = dbus.MakeVariant(restoreToken)
		}
		return []any{s.path, data}
	})
	if err != nil {
		return err
	}

	return nil
}

// Starts the screen cast session
func (s *Session) Start(parentWindow string) (string, []Stream, error) {
	return s.StartContext(context.Background(), parentWindow)
}

func (s *Session) StartContext(ctx context.Context, parentWindow string) (string, []Stream, error) {
	results, err := getResponse(ctx, "", startCallName, func(token string) []any {
		data := map[string]dbus.Variant{
			"handle_token": convert.FromString(token),
		}
		return []any{s.path, parentWindow, data}
	})
	if err != nil {
		return "", nil, err
	}

	restoreToken := ""
	rawToken, ok := results["restore_token"]
	if ok {
		restoreToken = rawToken.Value().(string)
	}

	rawStreams := results["streams"].Value().([][]interface{})
	streams := make([]Stream, len(rawStreams))
	for i, stream := range rawStreams {
		streams[i] = Stream{
			NodeId:     stream[0].(uint32),
			Properties: stream[1].(map[string]dbus.Variant),
		}
	}

	return restoreToken, streams, nil
}

// Open a file descriptor to the PipeWire remote where the screen cast streams are available
func (s *Session) OpenPipeWireRemote() (*os.File, error) {
	result, err := apis.Call(openPipeWireRemoteCallName, s.path, map[string]dbus.Variant{})
	if err != nil {
		return nil, err
	}
	fd, err := convert.UnixFDToUintptr(result.(dbus.UnixFD))
	if err != nil {
		return nil, err
	}

	return os.NewFile(fd, fmt.Sprintf("/proc/self/fd/%v", fd)), nil
}

func (s *Session) ensureDone() chan struct{} {
	if s.done == nil {
		s.done = make(chan struct{})
	}
	return s.done
}
