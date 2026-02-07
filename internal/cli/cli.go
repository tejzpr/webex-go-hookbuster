/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 */

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tejzpr/webex-go-hookbuster/internal/config"
	"github.com/tejzpr/webex-go-hookbuster/internal/display"
)

var scanner = bufio.NewScanner(os.Stdin)

// prompt prints a question and reads a line from stdin.
func prompt(question string) (string, error) {
	fmt.Print(display.Question(question))
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("EOF reached")
}

// RequestToken prompts the user for a Webex access token.
func RequestToken() (string, error) {
	answer, err := prompt("Enter your access token: ")
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", fmt.Errorf("token empty")
	}
	return answer, nil
}

// RequestTarget prompts the user for the forwarding target hostname/IP.
func RequestTarget() (string, error) {
	answer, err := prompt(`Enter a target you will forward messages to (e.g. "localhost"): `)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", fmt.Errorf("target empty")
	}
	return answer, nil
}

// RequestPort prompts the user for the forwarding port number.
func RequestPort() (int, error) {
	answer, err := prompt("Enter a port you will forward messages to: ")
	if err != nil {
		return 0, err
	}
	if answer == "" {
		return 0, fmt.Errorf("port empty")
	}
	port, err := strconv.Atoi(answer)
	if err != nil {
		return 0, fmt.Errorf("not a number")
	}
	return port, nil
}

// ResourceResult is returned by RequestResource.
// If AllResources is true, the user selected firehose mode ("all").
// Otherwise, Single contains the chosen resource.
type ResourceResult struct {
	AllResources bool
	Single       *config.Resource
}

// RequestResource prompts the user to select a resource.
// Returns ResourceResult indicating whether the user chose "all" or a
// single resource.
func RequestResource() (*ResourceResult, error) {
	// Build the prompt string showing aliases
	question := fmt.Sprintf(
		"Select resource [ a - all, %s - %s, %s - %s, %s - %s, %s - %s ]: ",
		config.Resources["rooms"].Alias, config.Resources["rooms"].Description,
		config.Resources["messages"].Alias, config.Resources["messages"].Description,
		config.Resources["memberships"].Alias, config.Resources["memberships"].Description,
		config.Resources["attachmentActions"].Alias, config.Resources["attachmentActions"].Description,
	)

	answer, err := prompt(question)
	if err != nil {
		return nil, err
	}
	if answer == "" {
		return nil, fmt.Errorf("response empty")
	}

	// Check for "all" alias
	if answer == "a" {
		return &ResourceResult{AllResources: true}, nil
	}

	// Find the resource by alias
	for _, res := range config.Resources {
		if res.Alias == answer {
			return &ResourceResult{Single: res}, nil
		}
	}

	return nil, fmt.Errorf("invalid selection")
}

// RequestEvent prompts the user to select an event from the given pool.
func RequestEvent(events []string) (string, error) {
	// Build option display
	var options []string
	for _, event := range events {
		alias := string(event[0])
		options = append(options, fmt.Sprintf("%s - %s", alias, event))
	}

	promptText := fmt.Sprintf("Select event [ %s ]: ", strings.Join(options, ", "))
	answer, err := prompt(promptText)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", fmt.Errorf("response empty")
	}

	// Match by first character alias
	for _, event := range events {
		if answer == string(event[0]) {
			fmt.Println(display.Answer(strings.ToUpper(event)))
			return event, nil
		}
	}

	return "", fmt.Errorf("event invalid")
}
