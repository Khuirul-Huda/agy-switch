package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/khuirulhuda/agy-switch/internal/adapter"
	"github.com/khuirulhuda/agy-switch/internal/domain"
	"github.com/khuirulhuda/agy-switch/internal/port"
	"github.com/khuirulhuda/agy-switch/internal/service"
)

const version = "0.1.0"
const projectURL = "https://github.com/Khuirul-Huda/agy-switch"
const passwordEnv = "AGY_SWITCH_PASSWORD"

var agyRunner = runAgy

func main() {
	keyringStorage := adapter.NewSystemKeyring(adapter.DefaultServiceName)
	ui := adapter.NewUI()
	locker, err := port.NewDefaultProcessLocker()
	if err != nil {
		ui.Error("Failed to initialize profile lock: %v", err)
		os.Exit(1)
	}
	profileService := service.NewProfileServiceWithLocker(keyringStorage, locker)

	args := os.Args[1:]
	if len(args) == 0 {
		if err := ui.RunInteractiveMenu(profileService); err != nil {
			ui.Error("Failed to run interactive menu: %v", err)
			os.Exit(1)
		}
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "current":
		if !handleCurrent(profileService, ui, hasFlag(args[1:], "--quiet"), hasFlag(args[1:], "--json")) {
			os.Exit(1)
		}
	case "save":
		handleSave(profileService, ui, args[1:])
	case "switch":
		handleSwitch(profileService, ui, args[1:])
	case "new":
		handleNewArgs(profileService, ui, agyRunner, args[1:])
	case "list", "ls":
		handleList(profileService, ui, hasFlag(args[1:], "--json"))
	case "doctor":
		if !handleDoctor(profileService, ui, hasFlag(args[1:], "--json")) {
			os.Exit(1)
		}
	case "rename":
		handleRename(profileService, ui, args[1:])
	case "export":
		handleExport(profileService, ui, args[1:])
	case "import":
		handleImport(profileService, ui, args[1:])
	case "completion":
		handleCompletion(profileService, ui, args[1:])
	case "delete", "rm":
		handleDelete(profileService, ui, args[1:])
	case "logout":
		handleLogout(profileService, ui)
	case "version", "-v", "--version":
		fmt.Printf("agy-switch v%s\n%s\nLicense: MIT License\n", version, projectURL)
	case "help", "-h", "--help":
		printUsage()
	default:
		if len(args) == 1 {
			handleSwitch(profileService, ui, args)
			return
		}
		ui.Error("Unknown command: '%s'. Run 'agy-switch help' for usage instructions.", subcommand)
		os.Exit(1)
	}
}

func handleCurrent(svc *service.ProfileService, ui *adapter.UI, quietFlags ...bool) bool {
	quiet := len(quietFlags) > 0 && quietFlags[0]
	asJSON := len(quietFlags) > 1 && quietFlags[1]
	current, err := svc.CurrentProfile()
	if err != nil {
		if asJSON {
			code := currentErrorCode(err)
			_ = ui.RenderJSON(map[string]any{
				"error": map[string]string{
					"code":    code,
					"message": currentErrorMessage(code),
				},
			})
		} else {
			ui.Error("%v", err)
		}
		return false
	}
	if asJSON {
		_ = ui.RenderJSON(map[string]string{"email": current.Email})
		return true
	}
	if !quiet {
		ui.Info("Active agy account: %s", current.Email)
	}
	return true
}

func currentErrorCode(err error) string {
	switch {
	case errors.Is(err, service.ErrNoActiveSession):
		return "no_active_session"
	case errors.Is(err, service.ErrInvalidRegistry):
		return "invalid_registry"
	default:
		return "keyring_error"
	}
}

func currentErrorMessage(code string) string {
	switch code {
	case "no_active_session":
		return "no active agy session found"
	case "invalid_registry":
		return "profile registry is invalid"
	default:
		return "unable to read the active agy session"
	}
}

func handleSave(svc *service.ProfileService, ui *adapter.UI, args []string) {
	name := firstNonFlag(args)
	if name == "" {
		ui.Error("Please provide a profile name. Example: agy-switch save work")
		return
	}
	force := hasFlag(args, "--force")
	var profile *domain.Profile
	var err error
	if force {
		profile, err = svc.SaveProfileForce(name)
	} else {
		profile, err = svc.SaveProfile(name)
	}
	if err != nil {
		ui.Error("Failed to save profile: %v", err)
		return
	}
	ui.Success("Profile '%s' (%s) successfully saved to OS Keyring!", profile.Name, profile.Email)
}

func handleSwitch(svc *service.ProfileService, ui *adapter.UI, args []string) {
	if len(args) < 1 {
		ui.Error("Please specify the profile name to activate. Example: agy-switch switch personal")
		return
	}
	name := args[0]
	profile, err := svc.SwitchProfile(name)
	if err != nil {
		ui.Error("Failed to switch profile: %v", err)
		return
	}
	ui.Success("Successfully switched to profile '%s' (%s)!", profile.Name, profile.Email)
}

func handleNew(svc *service.ProfileService, ui *adapter.UI, runAgy func() error) {
	handleNewArgs(svc, ui, runAgy, nil)
}

func handleNewArgs(svc *service.ProfileService, ui *adapter.UI, runAgy func() error, args []string) {
	var name string
	var err error
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
	} else {
		name, err = ui.PromptProfileName()
		if err != nil {
			ui.Error("Failed to read profile name: %v", err)
			return
		}
	}

	exists, err := svc.ProfileExists(name)
	if err != nil {
		ui.Error("Invalid profile name: %v", err)
		return
	}
	if exists {
		ui.Error("Profile '%s' already exists. Choose another name.", name)
		return
	}

	previous, hadPrevious, err := svc.CaptureActiveToken()
	if err != nil {
		ui.Error("Failed to inspect the active session: %v", err)
		return
	}
	if err := svc.LogoutActive(); err != nil {
		ui.Error("Failed to clear the active session: %v", err)
		return
	}

	ui.Info("Starting agy for a new sign-in...")
	for attempt := 1; attempt <= 3; attempt++ {
		if err := runAgy(); err == nil {
			current, currentExists, captureErr := svc.CaptureActiveToken()
			if captureErr != nil {
				_ = svc.RestoreActiveToken(previous, hadPrevious)
				ui.Error("Failed to inspect the new agy session: %v", captureErr)
				return
			}
			if !currentExists || (hadPrevious && current == previous) {
				_ = svc.RestoreActiveToken(previous, hadPrevious)
				ui.Error("agy exited without creating a new active session; previous session restored")
				return
			}
			break
		} else if attempt == 3 {
			_ = svc.RestoreActiveToken(previous, hadPrevious)
			ui.Error("agy sign-in failed after %d attempts; previous session restored", attempt)
			return
		} else {
			ui.Error("agy sign-in failed; the previous session will be restored if login is abandoned")
			retry, promptErr := ui.PromptRetry()
			if promptErr != nil || !retry {
				_ = svc.RestoreActiveToken(previous, hadPrevious)
				if promptErr != nil {
					ui.Error("failed to read retry choice; previous session restored")
				} else {
					ui.Info("Login cancelled; previous session restored")
				}
				return
			}
		}
	}

	profile, err := svc.SaveProfile(name)
	if err != nil {
		_ = svc.RestoreActiveToken(previous, hadPrevious)
		ui.Error("Failed to save the new profile: %v", err)
		return
	}
	ui.Success("Profile '%s' (%s) successfully created!", profile.Name, profile.Email)
}

func handleList(svc *service.ProfileService, ui *adapter.UI, jsonFlags ...bool) {
	asJSON := len(jsonFlags) > 0 && jsonFlags[0]
	profiles, err := svc.ListProfiles()
	if err != nil {
		ui.Error("Failed to list profiles: %v", err)
		return
	}

	if asJSON {
		if err := ui.RenderProfilesJSON(profiles); err != nil {
			ui.Error("Failed to render profiles: %v", err)
		}
		return
	}
	activeEmail := ""
	current, _ := svc.CurrentProfile()
	if current != nil {
		activeEmail = current.Email
	}
	ui.RenderProfilesTable(profiles, activeEmail)
}

func handleRename(svc *service.ProfileService, ui *adapter.UI, args []string) {
	if len(args) < 2 {
		ui.Error("Please provide old and new profile names. Example: agy-switch rename old new")
		return
	}
	if err := svc.RenameProfile(args[0], args[1]); err != nil {
		ui.Error("Failed to rename profile: %v", err)
		return
	}
	ui.Success("Profile '%s' renamed to '%s'.", args[0], args[1])
}

func handleExport(svc *service.ProfileService, ui *adapter.UI, args []string) {
	if len(args) < 1 {
		ui.Error("Please provide an export file path. Example: agy-switch export profiles.agy")
		return
	}
	password, err := readPassword(ui, "Export password: ")
	if err != nil {
		ui.Error("Failed to read export password: %v", err)
		return
	}
	if err := svc.ExportProfiles(port.OSFileSystem{}, args[0], password); err != nil {
		ui.Error("Failed to export profiles: %v", err)
		return
	}
	ui.Success("Encrypted profiles exported to '%s'.", args[0])
}

func handleImport(svc *service.ProfileService, ui *adapter.UI, args []string) {
	if len(args) < 1 {
		ui.Error("Please provide an encrypted export file path. Example: agy-switch import profiles.agy")
		return
	}
	password, err := readPassword(ui, "Import password: ")
	if err != nil {
		ui.Error("Failed to read import password: %v", err)
		return
	}
	if err := svc.ImportProfiles(port.OSFileSystem{}, args[0], password, hasFlag(args[1:], "--force")); err != nil {
		ui.Error("Failed to import profiles: %v", err)
		return
	}
	ui.Success("Encrypted profiles imported successfully.")
}

func readPassword(ui *adapter.UI, prompt string) (string, error) {
	if password := os.Getenv(passwordEnv); password != "" {
		return password, nil
	}
	return ui.PromptPassword(prompt)
}

func handleDoctor(svc *service.ProfileService, ui *adapter.UI, asJSON bool) bool {
	report := svc.Doctor(nil)
	if asJSON {
		if err := ui.RenderJSON(report); err != nil {
			ui.Error("Failed to render doctor report: %v", err)
		}
		return report.OK
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.OK {
			ui.Success("%s: %s", diagnostic.Check, diagnostic.Message)
		} else {
			ui.Error("%s: %s", diagnostic.Check, diagnostic.Message)
		}
	}
	if !report.OK {
		ui.Info("Run the failing checks above, then run 'agy-switch doctor' again.")
	}
	return report.OK
}

func handleCompletion(svc *service.ProfileService, ui *adapter.UI, args []string) {
	shell := "bash"
	if len(args) > 0 {
		shell = args[0]
	}
	switch shell {
	case "bash", "zsh", "fish", "powershell":
	default:
		ui.Error("Unsupported shell '%s'. Choose bash, zsh, fish, or powershell.", shell)
		return
	}
	profiles, err := svc.ListProfiles()
	if err != nil {
		ui.Error("Failed to load profiles for completion: %v", err)
		return
	}
	names := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		names = append(names, profile.Name)
	}
	ui.Println(completionScript(shell, names))
}

func completionScript(shell string, profiles []string) string {
	names := strings.Join(profiles, " ")
	switch shell {
	case "zsh":
		return fmt.Sprintf("#compdef agy-switch\n_arguments '1:command:(current save switch new list doctor delete rename logout export import completion %s)'", names)
	case "fish":
		return fmt.Sprintf("complete -c agy-switch -f -a 'current save switch new list doctor delete rename logout export import completion %s'", names)
	case "powershell":
		return fmt.Sprintf("Register-ArgumentCompleter -CommandName agy-switch -ScriptBlock { param($wordToComplete) '%s'.Split(' ') | Where-Object { $_ -like \"$wordToComplete*\" } }", strings.TrimSpace("current save switch new list doctor delete rename logout export import completion "+names))
	default:
		return fmt.Sprintf("_agy_switch_completions() { COMPREPLY=( $(compgen -W 'current save switch new list doctor delete rename logout export import completion %s' -- \"${COMP_WORDS[1]}\") ); }\ncomplete -F _agy_switch_completions agy-switch", names)
	}
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func firstNonFlag(args []string) string {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	return ""
}

func handleDelete(svc *service.ProfileService, ui *adapter.UI, args []string) {
	if len(args) < 1 {
		ui.Error("Please specify the profile name to delete. Example: agy-switch delete demo")
		return
	}
	name := args[0]
	if err := svc.DeleteProfile(name); err != nil {
		ui.Error("Failed to delete profile: %v", err)
		return
	}
	ui.Success("Profile '%s' successfully removed from OS Keyring.", name)
}

func handleLogout(svc *service.ProfileService, ui *adapter.UI) {
	if err := svc.LogoutActive(); err != nil {
		ui.Error("Failed to logout: %v", err)
		return
	}
	ui.Success("Active agy session cleared from OS Keyring.")
	fmt.Println("Launch 'agy' in your terminal to sign in with a new account.")
}

func printUsage() {
	fmt.Println("agy-switch - Multi-Account Profile Switcher for Antigravity CLI (agy)")
	fmt.Println("\nUsage:")
	fmt.Println("  agy-switch                      Open interactive menu to select and switch profiles")
	fmt.Println("  agy-switch current [--quiet|--json] Display the active account email")
	fmt.Println("  agy-switch save <name> [--force] Save the active session (never overwrites by default)")
	fmt.Println("  agy-switch switch <name>        Activate a saved profile for agy sessions")
	fmt.Println("  agy-switch <profile>            Shorthand for 'agy-switch switch <profile>'")
	fmt.Println("  agy-switch new [<name>]          Sign in with agy and save a new profile")
	fmt.Println("  agy-switch list [--json]         List profiles as a table or machine-readable JSON")
	fmt.Println("  agy-switch doctor [--json]       Check keyring, agy, registry, and profile health")
	fmt.Println("  agy-switch rename <old> <new>    Atomically rename a saved profile")
	fmt.Println("  agy-switch delete <name>        Delete a saved profile from the OS Keyring")
	fmt.Println("  agy-switch logout               Clear the active session to trigger a fresh agy sign-in")
	fmt.Println("  agy-switch export <file>         Encrypt profiles to a password-protected file")
	fmt.Println("  agy-switch import <file>         Import an encrypted profile export")
	fmt.Println("  agy-switch completion <shell>    Generate bash, zsh, fish, or powershell completion")
	fmt.Println("  agy-switch version              Show version information")
	fmt.Printf("\nProject: %s\nLicense: MIT License\n", projectURL)
}

func runAgy() error {
	cmd := exec.Command("agy")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
