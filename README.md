<div align="center">

# 🔄 agy-switch

**A sleek, secure, zero-overhead multi-account profile switcher for Antigravity CLI (`agy`).**

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platforms-Linux%20%7C%20macOS%20%7C%20Windows-555555?style=for-the-badge&logo=linux&logoColor=white)](https://github.com/Khuirul-Huda/agy-switch)
[![Architecture](https://img.shields.io/badge/Architecture-Clean%20Hexagonal-8A2BE2?style=for-the-badge)](https://github.com/Khuirul-Huda/agy-switch)
[![Tests](https://img.shields.io/badge/Tests-Hermetic%20In--Memory-brightgreen?style=for-the-badge&logo=githubactions&logoColor=white)](https://github.com/Khuirul-Huda/agy-switch)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)

Switch between Personal, Work, and Client Google accounts in `agy` instantly without opening the browser or re-authenticating every day.

---

📖 **[ 🇮🇩 Baca dalam Bahasa Indonesia ](README.id.md)**

---

</div>

> [!CAUTION]
> ### ⚠️ DISCLAIMER & NOTICE
> - **Use at your own risk:** This software is provided "as is", without warranty of any kind, express or implied. The author and contributors are not responsible for any potential issues, token revocations, or account-related actions resulting from the use of this software.
> - **Unofficial & Non-Affiliated:** `agy-switch` is an independent, community-developed open-source utility. It is **NOT affiliated with, sponsored by, authorized by, endorsed by, or in any way officially connected with Google LLC, Google DeepMind, Alphabet Inc., or any of their subsidiaries.**
> - "Google", "Antigravity", and "Gemini" are registered trademarks of Google LLC.

---

## 💡 The Problem

Google Antigravity CLI (`agy`) binds to a single Google account stored in your operating system's native secure keyring. When you need to alternate between your personal Google account and your organization's Google Cloud Workspace:
- You had to manually execute `/logout`.
- Wait for a browser window to pop up.
- Re-authenticate through Google Identity and click consent screens.
- Repeat the exact same flow every time you change tasks or projects.

## ✨ The Solution

`agy-switch` seamlessly manages named profiles directly inside your operating system's hardware-backed/encrypted credential store (**Linux Secret Service / KWallet / GNOME Keyring**, **macOS Apple Keychain**, and **Windows Credential Manager**).

- 🔒 **Zero Plaintext Files:** Tokens are never written to disk in plain JSON. Everything remains securely encrypted inside your OS Keyring.
- ⚡ **Instant Switch:** Swap active sessions in under 10 milliseconds.
- 🖥️ **Interactive & CLI Modes:** Supports standard subcommands for fast scripting as well as an interactive selection menu.
- 🌐 **True Cross-Platform:** Single Go binary compiled natively for Linux (`amd64`/`arm64`), macOS (`Apple Silicon`/`Intel`), and Windows (`amd64`).

---

## 🖥️ Demo Preview

```text
$ agy-switch

=== agy-switch : Antigravity Profile Switcher ===
Active account: personal@gmail.com

NO   PROFILE          EMAIL                            STATUS    
─────────────────────────────────────────────────────────────────
 1)  personal         personal@gmail.com               ● ACTIVE  
 2)  work             developer@company.com                      

Select profile number to switch (1-2, or 'q' to quit): 2
✓ Successfully switched to profile 'work' (developer@company.com)!
Launch 'agy' in a new terminal session to start working with this account.
```

---

## 🚀 Quick Start

### 1. Save your currently active `agy` session:
```bash
agy-switch save personal
```

### 2. Connect your second account (Work / School / Client):
```bash
agy-switch logout
agy   # agy will open your browser for a fresh Google Sign-in
```
Once signed in, save the new profile:
```bash
agy-switch save work
```

### 3. Switch anytime!
```bash
# Via interactive menu:
agy-switch

# Or directly via command:
agy-switch switch work
agy-switch switch personal

# Shorthand:
agy-switch work
```

### 4. Create a new profile
Run the onboarding flow to sign in with `agy` and save the resulting session:
```bash
agy-switch new
```
The command asks for a profile name, clears the active session, launches `agy` for interactive sign-in, and saves the new session after `agy` exits successfully. You can also use `agy-switch new work` in scripts. Failed sign-ins can be retried; abandoning the flow restores the previous session.

### 5. Export profiles securely
Exports are authenticated and encrypted with AES-GCM using a password-derived key. Tokens are never written as plaintext:
```bash
agy-switch export profiles.agy
agy-switch import profiles.agy
agy-switch import profiles.agy --force
```
Both commands prompt for the password. Export files are created with mode `0600`; keep the password separate from the file.

For CI or scripting, set `AGY_SWITCH_PASSWORD`. When this variable is non-empty it takes precedence over the prompt; it is never printed or included in JSON output. Environment variables can be visible to other processes on some systems, so prefer a protected secret store in CI.

### 6. Diagnose setup problems

```bash
agy-switch doctor
agy-switch doctor --json
```

The doctor checks OS keyring access, whether `agy` is available on `PATH`, registry validity, and every saved profile token. It reports remediation hints and never prints token contents.

---

## 📦 Installation

### Pre-compiled Binaries

Download the appropriate binary for your platform from the [Releases](https://github.com/Khuirul-Huda/agy-switch/releases) page:

#### Linux (Debian, Ubuntu, Arch, Fedora, openSUSE)
```bash
chmod +x agy-switch-linux-amd64
mv agy-switch-linux-amd64 ~/.local/bin/agy-switch
```

#### macOS (Apple Silicon M1/M2/M3/M4 & Intel)
```bash
# Apple Silicon:
chmod +x agy-switch-darwin-arm64
sudo mv agy-switch-darwin-arm64 /usr/local/bin/agy-switch

# Intel Mac:
chmod +x agy-switch-darwin-amd64
sudo mv agy-switch-darwin-amd64 /usr/local/bin/agy-switch
```

#### Windows
Rename `agy-switch-windows-amd64.exe` to `agy-switch.exe` and place it in your `PATH` (e.g., `C:\Windows\System32` or your user tools directory).

---

### Build From Source (Requires Go 1.26+)

```bash
git clone git@github.com:Khuirul-Huda/agy-switch.git
cd agy-switch

# Build for current host:
go build -o bin/agy-switch main.go

# Install locally:
cp bin/agy-switch ~/.local/bin/
```

#### Cross-Compilation Commands
```bash
# Linux 64-bit
GOOS=linux GOARCH=amd64 go build -o bin/agy-switch-linux-amd64 main.go

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o bin/agy-switch-darwin-arm64 main.go

# macOS Intel
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o bin/agy-switch-darwin-amd64 main.go

# Windows 64-bit
GOOS=windows GOARCH=amd64 go build -o bin/agy-switch-windows-amd64.exe main.go
```

---

## 📖 Commands Reference

| Command | Alias | Description |
|---|---|---|
| `agy-switch` | — | Opens interactive selection menu |
| `agy-switch current` | — | Shows the currently active account email |
| `agy-switch current --quiet` | — | Checks the active session without printing on success |
| `agy-switch current --json` | — | Emits stable JSON (`{"email":"..."}`) or a structured error; exits non-zero on failure |
| `agy-switch save <name>` | — | Saves the active token; refuses to overwrite an existing profile |
| `agy-switch save <name> --force` | — | Explicitly replaces an existing profile |
| `agy-switch switch <name>` | — | Activates a saved profile for `agy` sessions |
| `agy-switch <profile>` | — | Shorthand for `agy-switch switch <profile>` |
| `agy-switch new [<name>]` | — | Signs in with `agy` and saves the session as a new profile |
| `agy-switch list` | `ls` | Displays profiles and metadata in a table |
| `agy-switch list --json` | — | Emits metadata-only JSON (no token fields) |
| `agy-switch doctor [--json]` | — | Checks dependencies and profile integrity |
| `agy-switch rename <old> <new>` | — | Moves a profile token and metadata to a new name |
| `agy-switch delete <name>` | `rm` | Deletes a saved profile from the OS Keyring |
| `agy-switch logout` | — | Clears the active session to trigger a fresh sign-in |
| `agy-switch export <file>` | — | Creates a password-protected encrypted export |
| `agy-switch import <file>` | — | Imports an encrypted export; use `--force` to replace names |
| `agy-switch completion <shell>` | — | Generates bash, zsh, fish, or PowerShell completion |
| `agy-switch version` | `-v` | Displays current version |
| `agy-switch help` | `-h` | Shows usage information |

---

## 🔒 Security & Architecture

`agy-switch` interfaces with the exact same OS credential stores used by Google's `agy` binary:

```
┌────────────────────────────────────────────────────────┐
│                   agy-switch CLI                       │
└──────────────────────────┬─────────────────────────────┘
                           │
                 [ Keyring Storage Port ]
                           │
       ┌───────────────────┼───────────────────┐
       ▼                   ▼                   ▼
 Linux Secret Service  macOS Keychain   Windows Credential
 (KWallet / GNOME)    (Security API)         Manager
       │                   │                   │
       └───────────────────┼───────────────────┘
                           ▼
          Namespace: service = "gemini"
          • Active Token : username = "antigravity"
          • Saved Profile: username = "antigravity:profile:<name>"
          • Registry     : username = "antigravity:_profiles" (metadata only)
```

1. **Native OS Isolation:** Tokens are protected by your operating system login password / PAM module / biometric auth.
2. **Standard OAuth Refresh:** When switching to an older profile whose 1-hour access token has expired, `agy` uses the profile's embedded `refresh_token` to seamlessly fetch a fresh access token from Google servers.
3. **Hermetic & Safe:** Does not touch internal code databases or model caches, only the credential pointer read by `agy` on startup.
4. **Metadata without secrets:** The registry stores profile names and `created_at`/`last_used_at` timestamps. Tokens remain only in the OS keyring. Older registries containing a JSON array of names are migrated automatically.
5. **Safe concurrent writes:** profile and registry mutations use a per-user cross-platform process lock. Export files are written with a temporary file and atomic rename; imports validate the complete encrypted payload before changing the keyring and roll back partial writes.

---

## 🏛️ Clean Architecture & Testing

This project strictly adheres to **Clean Code**, **SOLID**, and **Ports & Adapters (Hexagonal Architecture)**:

```text
internal/
├── domain/            # Core business models & validation (Zero external dependencies)
│   ├── profile.go
│   └── profile_test.go
├── port/              # Contract interfaces
│   └── keyring.go     # KeyringStorage port
├── service/           # Application use cases (Save, Switch, List, Delete)
│   ├── profile_service.go
│   └── profile_service_test.go
└── adapter/           # Infrastructure & Delivery
    ├── system_keyring.go  # Real OS Keyring implementation
    ├── memory_keyring.go  # Hermetic In-Memory test double
    └── cli_ui.go          # ANSI terminal rendering & interactive prompt
```

### 🧪 Running Unit Tests (Hermetic In-Memory)
During automated tests, **the real system OS Keyring is NEVER touched**. All test suites execute against `MemoryKeyring` in RAM, guaranteeing safety and sub-millisecond execution:

```bash
go test -v -race ./...
```

The test suite injects an in-memory keyring, filesystem, clock, and password reader, so it does not require a real OS keyring or interactive terminal.

---

## 📄 License

Distributed under the **MIT License**. See `LICENSE` for more information.
