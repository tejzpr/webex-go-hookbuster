/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 */

package display

import "fmt"

// ANSI color codes
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	red     = "\033[31m"
	green   = "\033[32m"
	blue    = "\033[34m"
	magenta = "\033[35m"

	// colorFmt is the shared format string for wrapping text in a color.
	colorFmt = "%s%s%s"
)

// Question formats text as a bold prompt.
func Question(text string) string {
	return fmt.Sprintf(colorFmt+" ", bold, text, reset)
}

// Answer formats text as a green success message.
func Answer(text string) string {
	return fmt.Sprintf(colorFmt, green, text, reset)
}

// Info formats text as a blue informational message.
func Info(text string) string {
	return fmt.Sprintf(colorFmt, blue, text, reset)
}

// Error formats text as a red error message.
func Error(text string) string {
	return fmt.Sprintf(colorFmt, red, text, reset)
}

// Highlight formats text in magenta for emphasis.
func Highlight(text string) string {
	return fmt.Sprintf(colorFmt, magenta, text, reset)
}

// Welcome prints the hookbuster ASCII art banner.
func Welcome() {
	banner := `
  _   _             _    ____            _            
 | | | | ___   ___ | | _| __ ) _   _ ___| |_ ___ _ __ 
 | |_| |/ _ \ / _ \| |/ /  _ \| | | / __| __/ _ \ '__|
 |  _  | (_) | (_) |   <| |_) | |_| \__ \ ||  __/ |   
 |_| |_|\___/ \___/|_|\_\____/ \__,_|___/\__\___|_|   
                                                        
  GO Webex WebSocket-to-HTTP Event Bridge 
`
	fmt.Println(Info(banner))
}
