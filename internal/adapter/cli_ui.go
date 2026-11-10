package adapter

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/khuirulhuda/agy-switch/internal/domain"
	"github.com/khuirulhuda/agy-switch/internal/service"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorDim    = "\033[2m"
)

// UI provides formatted terminal output and interactive prompt handlers.
type UI struct {
	reader *bufio.Reader
}

// NewUI initializes a new terminal UI instance.
func NewUI() *UI {
	return &UI{
		reader: bufio.NewReader(os.Stdin),
	}
}

// Success prints a green success confirmation message.
func (u *UI) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✓ %s%s\n", colorGreen, msg, colorReset)
}

// Info prints a cyan informational message.
func (u *UI) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%sℹ %s%s\n", colorCyan, msg, colorReset)
}

// Error prints a red error message.
func (u *UI) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s✗ %s%s\n", colorRed, msg, colorReset)
}

// RenderProfilesTable displays formatted profile rows and marks the active profile.
func (u *UI) RenderProfilesTable(profiles []domain.Profile, activeEmail string) {
	if len(profiles) == 0 {
		fmt.Printf("%sNo saved profiles found.%s Use: %sagy-switch save <name>%s\n", colorDim, colorReset, colorBold, colorReset)
		return
	}

	fmt.Printf("\n%s%-4s %-16s %-32s %-10s%s\n", colorBold, "NO", "PROFILE", "EMAIL", "STATUS", colorReset)
	fmt.Println(strings.Repeat("─", 65))

	for idx, p := range profiles {
		status := ""
		badgeColor := colorDim
		if activeEmail != "" && p.Email == activeEmail {
			status = "● ACTIVE"
			badgeColor = colorGreen
		}

		fmt.Printf("%2d)  %-16s %-32s %s%-10s%s\n",
			idx+1,
			p.Name,
			p.Email,
			badgeColor,
			status,
			colorReset,
		)
	}
	fmt.Println()
}

// RunInteractiveMenu runs an interactive terminal selection when agy-switch is executed without subcommands.
func (u *UI) RunInteractiveMenu(svc *service.ProfileService) error {
	activeProfile, _ := svc.CurrentProfile()
	activeEmail := ""
	if activeProfile != nil {
		activeEmail = activeProfile.Email
	}

	fmt.Printf("\n%s%s=== agy-switch : Antigravity Profile Switcher ===%s\n", colorBold, colorCyan, colorReset)
	if activeEmail != "" {
		fmt.Printf("Active account: %s%s%s\n", colorBold, activeEmail, colorReset)
	} else {
		fmt.Printf("Active account: %s(Not signed in to agy)%s\n", colorYellow, colorReset)
	}

	profiles, err := svc.ListProfiles()
	if err != nil {
		return err
	}

	u.RenderProfilesTable(profiles, activeEmail)

	if len(profiles) == 0 {
		return nil
	}

	fmt.Printf("Select profile number to switch (%s1-%d%s, or %sq%s to quit): ",
		colorBold, len(profiles), colorReset, colorBold, colorReset)

	input, err := u.reader.ReadString('\n')
	if err != nil {
		return err
	}

	trimmed := strings.TrimSpace(input)
	if trimmed == "q" || trimmed == "exit" || trimmed == "" {
		fmt.Println("Cancelled.")
		return nil
	}

	choice, err := strconv.Atoi(trimmed)
	if err != nil || choice < 1 || choice > len(profiles) {
		u.Error("Invalid selection.")
		return nil
	}

	selected := profiles[choice-1]
	switched, err := svc.SwitchProfile(selected.Name)
	if err != nil {
		return err
	}

	u.Success("Successfully switched to profile '%s' (%s)!", switched.Name, switched.Email)
	fmt.Printf("%sLaunch 'agy' in a new terminal session to start working with this account.%s\n\n", colorDim, colorReset)
	return nil
}
