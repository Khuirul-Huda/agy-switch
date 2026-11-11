package adapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

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
	reader         *bufio.Reader
	out            io.Writer
	errOut         io.Writer
	passwordReader func() ([]byte, error)
}

// NewUI initializes a new terminal UI instance.
func NewUI() *UI {
	return NewUIWithReader(os.Stdin)
}

// NewUIWithReader initializes a terminal UI with a custom input source.
func NewUIWithReader(reader io.Reader) *UI {
	return NewUIWithWriters(reader, os.Stdout, os.Stderr)
}

// NewUIWithWriters initializes a terminal UI with injectable input/output.
func NewUIWithWriters(reader io.Reader, out, errOut io.Writer) *UI {
	ui := &UI{
		reader: bufio.NewReader(reader),
		out:    out,
		errOut: errOut,
	}
	if file, ok := reader.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		ui.passwordReader = func() ([]byte, error) {
			return term.ReadPassword(int(file.Fd()))
		}
	}
	return ui
}

// SetPasswordReader injects a password source for tests and non-terminal callers.
func (u *UI) SetPasswordReader(reader func() ([]byte, error)) {
	u.passwordReader = reader
}

// Success prints a green success confirmation message.
func (u *UI) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(u.out, "%s✓ %s%s\n", colorGreen, msg, colorReset)
}

// Info prints a cyan informational message.
func (u *UI) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(u.out, "%sℹ %s%s\n", colorCyan, msg, colorReset)
}

// Error prints a red error message.
func (u *UI) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(u.errOut, "%s✗ %s%s\n", colorRed, msg, colorReset)
}

func (u *UI) Println(value string) {
	fmt.Fprintln(u.out, value)
}

func (u *UI) Printf(format string, args ...any) {
	fmt.Fprintf(u.out, format, args...)
}

// PromptProfileName reads and trims a profile name from the terminal.
func (u *UI) PromptProfileName() (string, error) {
	fmt.Fprint(u.out, "Enter a name for the new profile: ")
	input, err := u.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func (u *UI) PromptRetry() (bool, error) {
	fmt.Fprint(u.out, "Retry login? [y/N]: ")
	input, err := u.reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (u *UI) PromptPassword(label string) (string, error) {
	fmt.Fprint(u.out, label)
	if u.passwordReader != nil {
		input, err := u.passwordReader()
		fmt.Fprintln(u.out)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(input)), nil
	}
	input, err := u.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// RenderProfilesTable displays formatted profile rows and marks the active profile.
func (u *UI) RenderProfilesTable(profiles []domain.Profile, activeEmail string) {
	if len(profiles) == 0 {
		fmt.Fprintf(u.out, "%sNo saved profiles found.%s Use: %sagy-switch save <name>%s\n", colorDim, colorReset, colorBold, colorReset)
		return
	}

	fmt.Fprintf(u.out, "\n%s%-4s %-16s %-28s %-10s %-20s %-20s%s\n", colorBold, "NO", "PROFILE", "EMAIL", "STATUS", "CREATED", "LAST USED", colorReset)
	fmt.Fprintln(u.out, strings.Repeat("-", 116))

	for idx, p := range profiles {
		status := ""
		badgeColor := colorDim
		if p.Active {
			status = "● ACTIVE"
			badgeColor = colorGreen
		}

		created := formatMetadataTime(p.CreatedAt)
		lastUsed := formatMetadataTime(p.LastUsedAt)
		fmt.Fprintf(u.out, "%2d)  %-16s %-28s %s%-10s%s %-20s %-20s\n",
			idx+1,
			p.Name,
			p.Email,
			badgeColor,
			status,
			colorReset,
			created,
			lastUsed,
		)
	}

	fmt.Fprintln(u.out)
}

func formatMetadataTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func (u *UI) RenderProfilesJSON(profiles []domain.Profile) error {
	encoded, err := json.Marshal(profiles)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(u.out, string(encoded))
	return err
}

func (u *UI) RenderJSON(value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(u.out, string(encoded))
	return err
}

// RunInteractiveMenu runs an interactive terminal selection when agy-switch is executed without subcommands.
func (u *UI) RunInteractiveMenu(svc *service.ProfileService) error {
	activeProfile, _ := svc.CurrentProfile()
	activeEmail := ""
	if activeProfile != nil {
		activeEmail = activeProfile.Email
	}

	fmt.Fprintf(u.out, "\n%s%s=== agy-switch : Antigravity Profile Switcher ===%s\n", colorBold, colorCyan, colorReset)
	if activeEmail != "" {
		fmt.Fprintf(u.out, "Active account: %s%s%s\n", colorBold, activeEmail, colorReset)
	} else {
		fmt.Fprintf(u.out, "Active account: %s(Not signed in to agy)%s\n", colorYellow, colorReset)
	}

	profiles, err := svc.ListProfiles()
	if err != nil {
		return err
	}

	u.RenderProfilesTable(profiles, activeEmail)

	if len(profiles) == 0 {
		return nil
	}

	fmt.Fprintf(u.out, "Select profile number to switch (%s1-%d%s, or %sq%s to quit): ",
		colorBold, len(profiles), colorReset, colorBold, colorReset)

	input, err := u.reader.ReadString('\n')
	if err != nil {
		return err
	}

	trimmed := strings.TrimSpace(input)
	if trimmed == "q" || trimmed == "exit" || trimmed == "" {
		fmt.Fprintln(u.out, "Cancelled.")
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
	fmt.Fprintf(u.out, "%sLaunch 'agy' in a new terminal session to start working with this account.%s\n\n", colorDim, colorReset)
	return nil
}
