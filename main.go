package main

import (
	"fmt"
	"os"

	"github.com/khuirulhuda/agy-switch/internal/adapter"
	"github.com/khuirulhuda/agy-switch/internal/service"
)

const version = "0.0.1"

func main() {
	keyringStorage := adapter.NewSystemKeyring(adapter.DefaultServiceName)
	profileService := service.NewProfileService(keyringStorage)
	ui := adapter.NewUI()

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
		handleCurrent(profileService, ui)
	case "save":
		handleSave(profileService, ui, args[1:])
	case "switch":
		handleSwitch(profileService, ui, args[1:])
	case "list", "ls":
		handleList(profileService, ui)
	case "delete", "rm":
		handleDelete(profileService, ui, args[1:])
	case "logout":
		handleLogout(profileService, ui)
	case "version", "-v", "--version":
		fmt.Printf("agy-switch v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		ui.Error("Unknown command: '%s'. Run 'agy-switch help' for usage instructions.", subcommand)
		os.Exit(1)
	}
}

func handleCurrent(svc *service.ProfileService, ui *adapter.UI) {
	current, err := svc.CurrentProfile()
	if err != nil {
		ui.Error("%v", err)
		return
	}
	ui.Info("Active agy account: %s", current.Email)
}

func handleSave(svc *service.ProfileService, ui *adapter.UI, args []string) {
	if len(args) < 1 {
		ui.Error("Please provide a profile name. Example: agy-switch save work")
		return
	}
	name := args[0]
	profile, err := svc.SaveProfile(name)
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

func handleList(svc *service.ProfileService, ui *adapter.UI) {
	profiles, err := svc.ListProfiles()
	if err != nil {
		ui.Error("Failed to list profiles: %v", err)
		return
	}

	activeEmail := ""
	current, _ := svc.CurrentProfile()
	if current != nil {
		activeEmail = current.Email
	}

	ui.RenderProfilesTable(profiles, activeEmail)
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
	fmt.Println("  agy-switch current              Display the email address of the currently active account")
	fmt.Println("  agy-switch save <name>          Save the active agy session into a named profile")
	fmt.Println("  agy-switch switch <name>        Activate a saved profile for agy sessions")
	fmt.Println("  agy-switch list                 List all saved profiles in a formatted table")
	fmt.Println("  agy-switch delete <name>        Delete a saved profile from the OS Keyring")
	fmt.Println("  agy-switch logout               Clear the active session to trigger a fresh agy sign-in")
	fmt.Println("  agy-switch version              Show version information")
}
