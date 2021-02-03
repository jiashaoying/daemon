package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"text/template"
)

type supervisord struct {
	c *config
}

func (s *supervisord) Install() (err error) {
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

	tpl, err := template.New("supervisordScript").Parse(supervisordScript)
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

	if err := s.configLogFile(); err != nil {
		_ = os.Remove(s.servicePath())
		return err
	}

	if err := exec.Command("supervisorctl", "reread").Run(); err != nil {
		_ = os.Remove(s.servicePath())
		return err
	}
	if err := exec.Command("supervisorctl", "add", s.c.Name).Run(); err != nil {
		_ = os.Remove(s.servicePath())
		return err
	}
	return nil
}

func (s *supervisord) Enable() error {
	return nil
}

func (s *supervisord) Disable() error {
	return nil
}

func (s *supervisord) Remove() error {
	if err := checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	_ = s.Stop()
	_ = exec.Command("supervisorctl", "remove", s.c.Name).Run()
	_ = os.Remove(s.servicePath())
	_ = exec.Command("supervisorctl", "reread").Run()
	return nil
}

func (s *supervisord) Start() error {
	if err := checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	if s.isRunning() {
		return errAlreadyRunning
	}
	if err := s.configLogFile(); err != nil {
		return err
	}

	if err := exec.Command("supervisorctl", "start", s.c.Name).Run(); err != nil {
		return err
	}
	return nil
}

func (s *supervisord) Stop() error {
	if err := checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	if !s.isRunning() {
		return errAlreadyStopped
	}

	if err := exec.Command("supervisorctl", "stop", s.c.Name).Run(); err != nil {
		return err
	}
	return nil
}

func (s *supervisord) Status() error {
	if err := checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	output, err := exec.Command("supervisorctl", "status", s.c.Name).Output()
	if err == nil {
		reg := regexp.MustCompile("(STARTING|RUNNING|STOPPED)")
		stat := reg.FindString(string(output))
		if stat == "STARTING" {
			fmt.Println("Service is starting...")
		} else if stat == "RUNNING" {
			reg := regexp.MustCompile("pid ([0-9]+)")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				fmt.Println("Service (pid " + data[1] + ") is running")
			} else {
				fmt.Println("Service is running")
			}
		} else if stat == "STOPPED" {
			fmt.Println("Service has stopped")
		}
		return nil
	}
	fmt.Println("Service has stopped")
	return nil
}

func (s *supervisord) Log() error {
	if !s.isInstalled() {
		return errNotInstalled
	}
	if err := s.configLogFile(); err != nil {
		return err
	}
	return execCommandWithOutput("supervisorctl", "tail", "-f", s.c.Name)
}

func (s *supervisord) servicePath() string {
	return "/etc/supervisor/conf.d/" + s.c.Name + ".ini"
}

func (s *supervisord) isInstalled() bool {
	if _, err := os.Stat(s.servicePath()); err != nil {
		return false
	}
	return true
}

func (s *supervisord) isRunning() bool {
	output, err := exec.Command("supervisorctl", "status", s.c.Name).Output()
	if err != nil {
		return false
	}
	reg := regexp.MustCompile("(RUNNING|STARTING)")
	return reg.MatchString(string(output))
}

func (s *supervisord) configLogFile() error {
	logPath := "/var/log/" + s.c.Name + "/"
	logFile := logPath + s.c.Name + ".log"
	if pathOrFileIsExist(logFile) {
		return nil
	}
	if !pathOrFileIsExist(logPath) {
		if err := os.MkdirAll(logPath, 0755); err != nil {
			return err
		}
	}
	file, err := os.Create(logFile)
	if err != nil {
		return err
	}
	return file.Close()
}

var supervisordScript = `[program:{{.Name}}]
directory={{.WorkDir}}
command={{.Exec}} {{.Args}}
autostart=true
autorestart=unexpected
exitcodes=0
redirect_stderr=true
stdout_logfile_maxbytes=50MB
stdout_logfile_backups=10
stdout_logfile={{.LogFile}}
`
