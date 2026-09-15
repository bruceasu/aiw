// Package notification adapts the durable Workflow outbox to the Python
// aiw-notify Plugin protocol.
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"aiw/internal/plugin"
	"aiw/internal/workflow"
)

const notifyPluginName = "notify"

// PluginDispatcher dispatches persisted outbox records through aiw-notify.
// Plugin discovery remains centralized in the existing AIW plugin mechanism.
type PluginDispatcher struct{}

type pluginResponse struct {
	NotificationID string `json:"notification_id"`
	Status         string `json:"status"`
	Receipt        string `json:"receipt"`
}

func (PluginDispatcher) Dispatch(ctx context.Context, notification workflow.Notification) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := plugin.DiscoverPlugin(notifyPluginName)
	if err != nil {
		return "", fmt.Errorf("discover aiw-notify Plugin: %w", err)
	}
	payload, err := json.Marshal(notification)
	if err != nil {
		return "", fmt.Errorf("encode notification: %w", err)
	}
	output, code, err := plugin.ExecPluginWithInput(path, []string{"dispatch", "--json"}, map[string]string{
		"AIW_NOTIFICATION_ID": string(notification.ID),
	}, payload)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("aiw-notify Plugin exited with code %d", code)
	}
	var response pluginResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return "", fmt.Errorf("decode aiw-notify Plugin response: %w", err)
	}
	if response.NotificationID != string(notification.ID) || response.Status != "accepted" || strings.TrimSpace(response.Receipt) == "" {
		return "", fmt.Errorf("aiw-notify Plugin returned an invalid dispatch receipt")
	}
	return response.Receipt, nil
}
