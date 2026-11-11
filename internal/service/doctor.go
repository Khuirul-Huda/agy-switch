package service

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"

	"github.com/khuirulhuda/agy-switch/internal/domain"
	"github.com/khuirulhuda/agy-switch/internal/port"
)

type Diagnostic struct {
	Check   string `json:"check"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type DoctorReport struct {
	OK          bool         `json:"ok"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

const doctorProbeKey = "antigravity:_agy_switch_doctor_probe"

// Doctor checks dependencies and every registered profile without exposing tokens.
func (s *ProfileService) Doctor(agyLookup func() (string, error)) DoctorReport {
	unlock, err := s.acquireLock()
	if err != nil {
		return DoctorReport{
			OK: false,
			Diagnostics: []Diagnostic{{
				Check: "lock", OK: false, Message: "profile lock is unavailable",
			}},
		}
	}
	defer unlock()

	report := DoctorReport{OK: true, Diagnostics: []Diagnostic{}}
	add := func(check string, ok bool, message string) {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{Check: check, OK: ok, Message: message})
		if !ok {
			report.OK = false
		}
	}

	if _, err := s.storage.Get(doctorProbeKey); err == nil {
		add("keyring", true, "OS keyring is accessible")
	} else if errors.Is(err, port.ErrNotFound) {
		if setErr := s.storage.Set(doctorProbeKey, "agy-switch-doctor-probe"); setErr != nil {
			add("keyring", false, "OS keyring is unavailable; check the OS credential service")
		} else if deleteErr := s.storage.Delete(doctorProbeKey); deleteErr != nil {
			add("keyring", false, "OS keyring is readable but the diagnostic write could not be removed")
		} else {
			add("keyring", true, "OS keyring is readable and writable")
		}
	} else {
		add("keyring", false, "OS keyring is unavailable; check the OS credential service")
	}

	if agyLookup == nil {
		agyLookup = func() (string, error) { return exec.LookPath("agy") }
	}
	if path, err := agyLookup(); err != nil {
		add("agy", false, "agy executable was not found on PATH; install agy or adjust PATH")
	} else {
		add("agy", true, "agy executable found at "+path)
	}

	registry, err := s.loadRegistry()
	if err != nil {
		add("registry", false, "profile registry is invalid; remove or repair the registry key")
		return report
	}
	add("registry", true, fmt.Sprintf("profile registry is valid (%d profiles)", len(registry.Profiles)))

	names := make([]string, 0, len(registry.Profiles))
	for name := range registry.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		raw, getErr := s.storage.Get(formatProfileKey(name))
		if errors.Is(getErr, port.ErrNotFound) {
			add("profile:"+name, false, "profile token is missing from the keyring")
			continue
		}
		if getErr != nil {
			add("profile:"+name, false, "profile token could not be read from the keyring")
			continue
		}
		if _, parseErr := domain.ParseTokenPayload(raw); parseErr != nil {
			add("profile:"+name, false, "profile token is malformed")
			continue
		}
		add("profile:"+name, true, "profile token is present and valid")
	}
	return report
}
