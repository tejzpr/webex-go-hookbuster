/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 */

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/tejzpr/webex-go-hookbuster/internal/cli"
	"github.com/tejzpr/webex-go-hookbuster/internal/config"
	"github.com/tejzpr/webex-go-hookbuster/internal/display"
	"github.com/tejzpr/webex-go-hookbuster/internal/listener"
)

func main() {
	tokenEnv := os.Getenv("TOKEN")
	portEnv := os.Getenv("PORT")

	if tokenEnv != "" && portEnv != "" {
		// ── Deployment mode (environment variables) ──────────────────────
		runDeploymentMode(tokenEnv, portEnv)
	} else {
		// ── Interactive mode (CLI prompts) ───────────────────────────────
		display.Welcome()
		runInteractiveMode()
	}
}

// runDeploymentMode uses environment variables to configure hookbuster and
// subscribes to ALL resources with ALL events (firehose mode).
func runDeploymentMode(token, portStr string) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println(display.Error("PORT is not a valid number"))
		os.Exit(1)
	}

	target := os.Getenv("TARGET")
	if target == "" {
		target = "localhost"
	}

	// Verify the token
	person, err := listener.VerifyAccessToken(token)
	if err != nil {
		fmt.Println(display.Error(err.Error()))
		os.Exit(1)
	}
	fmt.Println(display.Info(fmt.Sprintf("token authenticated as %s", person.DisplayName)))
	fmt.Println(display.Info(fmt.Sprintf("forwarding target set as %s", target)))

	specs := &config.Specs{
		Target:      target,
		AccessToken: token,
		Port:        port,
		Selection: config.Selection{
			Event: "all",
		},
	}

	l, err := listener.NewListener(specs)
	if err != nil {
		fmt.Println(display.Error(err.Error()))
		os.Exit(1)
	}

	// Register all firehose resources
	for _, resName := range config.FirehoseResourceNames {
		res := config.Resources[resName]
		if err := l.Start(res, "all"); err != nil {
			fmt.Println(display.Error(err.Error()))
			os.Exit(1)
		}
	}

	// Wait for SIGINT / SIGTERM
	waitForShutdown(l)
}

// runInteractiveMode guides the user through CLI prompts.
func runInteractiveMode() {
	specs := &config.Specs{}

	// 1. Gather access token (retry on failure)
	gatherToken(specs)

	// 2. Gather target
	gatherTarget(specs)

	// 3. Gather port
	gatherPort(specs)

	// 4. Gather resource (and event)
	l := gatherResourceAndStart(specs)

	// Wait for SIGINT / SIGTERM
	waitForShutdown(l)
}

// ── Interactive step functions ──────────────────────────────────────────

func gatherToken(specs *config.Specs) {
	for {
		token, err := cli.RequestToken()
		if err != nil {
			fmt.Println(display.Error(err.Error()))
			continue
		}

		person, err := listener.VerifyAccessToken(token)
		if err != nil {
			fmt.Println(display.Error(err.Error()))
			continue
		}

		fmt.Println(display.Info(fmt.Sprintf("token authenticated as %s", person.DisplayName)))
		specs.AccessToken = token
		return
	}
}

func gatherTarget(specs *config.Specs) {
	for {
		target, err := cli.RequestTarget()
		if err != nil {
			fmt.Println(display.Error(err.Error()))
			continue
		}

		if len(target) == 0 {
			fmt.Println(display.Error(`not a valid target. target must be "localhost", a valid IP address, or hostname`))
			continue
		}

		fmt.Println(display.Answer(target))
		specs.Target = target
		return
	}
}

func gatherPort(specs *config.Specs) {
	for {
		port, err := cli.RequestPort()
		if err != nil {
			fmt.Println(display.Error(err.Error()))
			continue
		}

		fmt.Println(display.Answer(fmt.Sprintf("%d", port)))
		specs.Port = port
		return
	}
}

func gatherResourceAndStart(specs *config.Specs) *listener.Listener {
	for {
		result, err := cli.RequestResource()
		if err != nil {
			fmt.Println(display.Error(err.Error()))
			continue
		}

		l := createListener(specs)

		if result.AllResources {
			startFirehose(l, specs)
			return l
		}

		startSingleResource(l, specs, result.Single)
		return l
	}
}

func createListener(specs *config.Specs) *listener.Listener {
	l, err := listener.NewListener(specs)
	if err != nil {
		fmt.Println(display.Error(err.Error()))
		os.Exit(1)
	}
	return l
}

func startFirehose(l *listener.Listener, specs *config.Specs) {
	specs.Selection.Event = "all"
	for _, resName := range config.FirehoseResourceNames {
		res := config.Resources[resName]
		if err := l.Start(res, "all"); err != nil {
			fmt.Println(display.Error(err.Error()))
			os.Exit(1)
		}
	}
}

func startSingleResource(l *listener.Listener, specs *config.Specs, res *config.Resource) {
	fmt.Println(display.Answer(res.Description))
	specs.Selection.Resource = res.Description

	event := gatherEvent(res)
	specs.Selection.Event = event

	if err := l.Start(res, event); err != nil {
		fmt.Println(display.Error(err.Error()))
		os.Exit(1)
	}
}

func gatherEvent(resource *config.Resource) string {
	for {
		event, err := cli.RequestEvent(resource.Events)
		if err != nil {
			fmt.Println(display.Error(err.Error()))
			continue
		}
		return event
	}
}

// ── Shutdown ────────────────────────────────────────────────────────────

func waitForShutdown(l *listener.Listener) {
	fmt.Println(display.Info("Press Ctrl+C to exit."))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println()
	if err := l.Stop(); err != nil {
		fmt.Println(display.Error(fmt.Sprintf("error stopping listener: %s", err.Error())))
	}
}
