/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 */

package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tejzpr/webex-go-hookbuster/internal/config"
	"github.com/tejzpr/webex-go-hookbuster/internal/display"
)

// Forward sends a webhook event as an HTTP POST request to the target.
func Forward(target string, port int, event config.WebhookEvent) error {
	// Serialize the event to pretty-printed JSON (matches Node.js behaviour)
	data, err := json.MarshalIndent(event, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d", target, port)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(display.Error(fmt.Sprintf("forward error: %s", err.Error())))
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("statusCode: %d\n", resp.StatusCode)
	fmt.Println(display.Info(fmt.Sprintf("event forwarded to %s:%d", target, port)))
	fmt.Println(display.Info(string(data)))

	return nil
}
