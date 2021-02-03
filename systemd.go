package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

type systemd struct {
	c *config
}

func (s *systemd) Install() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to install: %w", err)
		}
	}()
	if err = checkPrivileges(); err != nil {
		return err
	}

	if s.isInstalled() {
		return errAlreadyInstalled
	}

	if s.c.Exec, err = executablePath(s.c.Exec); err != nil {
		return err
	}

	tpl, err := template.New("systemdScript").Parse(systemdScript)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(s.servicePath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := tpl.Execute(file, s.c); err != nil {
		_ = os.Remove(s.servicePath())
		return err
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		_ = os.Remove(s.servicePath())
		return err
	}

	if err := exec.Command("systemctl", "enable", s.c.Name+".service").Run(); err != nil {
		_ = os.Remove(s.servicePath())
		return err
	}

	return nil
}

func (s *systemd) Enable() error {
	return nil
}

func (s *systemd) Disable() error {
	return nil
}

func (s *systemd) Remove() error {
	if err := checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}

	_ = s.Stop()

	_ = exec.Command("systemctl", "disable", s.c.Name+".service").Run()

	_ = os.Remove(s.servicePath())

	return nil
}

func (s *systemd) Start() error {
	if err := checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	if s.isRunning() {
		return errAlreadyRunning
	}

	if err := exec.Command("systemctl", "start", s.c.Name).Run(); err != nil {
		return err
	}

	return nil
}

func (s *systemd) Stop() error {
	if err := checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	if !s.isRunning() {
		return errAlreadyStopped
	}

	if err := exec.Command("systemctl", "stop", s.c.Name).Run(); err != nil {
		return err
	}

	return nil
}

func (s *systemd) Status() error {
	if err := checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	output, err := exec.Command("systemctl", "status", s.c.Name+".service").Output()
	if err == nil {
		if matched, err := regexp.MatchString("Active: active", string(output)); err == nil && matched {
			reg := regexp.MustCompile("Main PID: ([0-9]+)")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				fmt.Println("Service (pid " + data[1] + ") is running")
			} else {
				fmt.Println("Service is running")
			}
			return nil
		}
	}
	fmt.Println("Service has stopped")
	return nil
}

func (s *systemd) Log() error {
	if !s.isInstalled() {
		return errNotInstalled
	}
	fmt.Println("==> Press Ctrl-C to exit <==")
	return execCommandWithOutput("journalctl", "-fu", s.c.Name)
}

func (s *systemd) servicePath() string {
	return "/etc/systemd/system/" + s.c.Name + ".service"
}

func (s *systemd) isInstalled() bool {
	if _, err := os.Stat(s.servicePath()); err != nil {
		return false
	}
	return true
}

func (s *systemd) isRunning() bool {
	output, err := exec.Command("systemctl", "is-active", s.c.Name+".service").Output()
	if err == nil {
		reg := regexp.MustCompile("active")
		return reg.MatchString(strings.ToLower(string(output)))
	}
	return false
}

var systemdScript = `[Unit]
Description={{.Description}}
{{- $deps := "network-online.target local-fs.target time-sync.target nss-lookup.target"}}
{{- if .Dependencies}}
{{$deps = .Dependencies}}
{{- end}}
Requires={{$deps}}
After={{$deps}}

[Service]
User={{.User}}
StartLimitInterval=5
StartLimitBurst=10
WorkingDirectory={{.WorkDir}}
PIDFile=/var/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /var/run/{{.Name}}.pid
ExecStart={{.Exec}} {{.Args}}
Restart=on-failure
RestartSec=30

[Install]
WantedBy=default.target
`
