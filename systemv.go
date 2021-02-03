package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"text/template"
)

type systemv struct {
	c *config
}

func (s *systemv) Install() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to install service: %w", err)
		}
	}()
	if err = checkPrivileges(); err != nil {
		return err
	}

	if s.isInstalled() {
		return errAlreadyInstalled
	}

	s.c.Exec, err = executablePath(s.c.Exec)
	if err != nil {
		return err
	}

	tpl, err := template.New("systemvScript").Parse(systemvScript)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(s.servicePath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	// clean up the service file if an error occurs in the next operation
	defer func() {
		if err != nil {
			_ = os.Remove(s.servicePath())
		}
	}()
	defer file.Close()

	if err = tpl.Execute(file, s.c); err != nil {
		return err
	}

	if err = s.configLogRoate(); err != nil {
		return err
	}

	if err = s.enable(); err != nil {
		return err
	}
	return
}

func (s *systemv) Enable() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to enable service: %w", err)
		}
	}()
	if err = checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	if err = s.enable(); err != nil {
		return err
	}
	return
}

func (s *systemv) enable() error {
	if err := exec.Command("chkconfig", "--add", s.c.Name).Run(); err != nil {
		return err
	}
	return nil
}

func (s *systemv) Disable() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to disable service: %w", err)
		}
	}()
	if err = checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	if err = s.disable(); err != nil {
		return err
	}
	return nil
}

func (s *systemv) disable() error {
	if err := exec.Command("chkconfig", "--del", s.c.Name).Run(); err != nil {
		return err
	}
	return nil
}

func (s *systemv) Remove() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to remove service: %w", err)
		}
	}()
	if err = checkPrivileges(); err != nil {
		return err
	}
	if !s.isInstalled() {
		return errNotInstalled
	}
	if err = s.stop(); err != nil {
		return err
	}
	if err = s.disable(); err != nil {
		return err
	}
	if err = os.Remove(s.servicePath()); err != nil {
		return err
	}
	if err = s.removeLogRoateConf(); err != nil {
		return err
	}
	return
}

func (s *systemv) Start() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to start service: %w", err)
		}
	}()
	if !s.isInstalled() {
		return errNotInstalled
	}
	if s.isRunning() {
		return errAlreadyRunning
	}
	if err = s.configLogFile(); err != nil {
		return err
	}

	if err = exec.Command("service", s.c.Name, "start").Run(); err != nil {
		return err
	}
	return
}

func (s *systemv) Stop() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to stop service: %w", err)
		}
	}()
	if !s.isInstalled() {
		return errNotInstalled
	}
	if !s.isRunning() {
		return errAlreadyStopped
	}

	if err = s.stop(); err != nil {
		return err
	}
	return
}

func (s *systemv) stop() (err error) {
	if err := exec.Command("service", s.c.Name, "stop").Run(); err != nil {
		return err
	}
	return
}

func (s *systemv) Status() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to show service's status: %w", err)
		}
	}()
	if !s.isInstalled() {
		return errNotInstalled
	}
	output, err := exec.Command("service", s.c.Name, "status").Output()
	if err != nil {
		fmt.Println("service has stopped")
		return nil
	}
	matched, err := regexp.MatchString(s.c.Name, string(output))
	if err != nil {
		return err
	}

	if matched {
		reg := regexp.MustCompile("pid  ([0-9]+)")
		data := reg.FindStringSubmatch(string(output))
		if len(data) > 1 {
			fmt.Println("service (pid " + data[1] + ") is running")
		} else {
			fmt.Println("service is running")
		}
	} else {
		fmt.Println(string(output))
	}
	return
}

func (s *systemv) Log() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to show service's log: %w", err)
		}
	}()
	if !s.isInstalled() {
		return errNotInstalled
	}
	if err = s.configLogFile(); err != nil {
		return err
	}
	fmt.Println("==> Press Ctrl-C to exit <==")
	_ = execCommandWithOutput("tail", "-f", s.c.LogFile)
	return
}

func (s *systemv) servicePath() string {
	return "/etc/init.d/" + s.c.Name
}

func (s *systemv) isInstalled() bool {
	if _, err := os.Stat(s.servicePath()); err != nil {
		return false
	}
	return true
}

func (s *systemv) isRunning() bool {
	output, err := exec.Command("service", s.c.Name, "status").Output()
	if err == nil {
		if matched, err := regexp.MatchString(s.c.Name, string(output)); err == nil && matched {
			return true
		}
	}
	return false
}

func (s *systemv) configLogFile() (err error) {
	if pathOrFileIsExist(s.c.LogFile) {
		return
	}
	logPath := path.Dir(s.c.LogFile)
	if !pathOrFileIsExist(logPath) {
		if err = os.MkdirAll(logPath, 0755); err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_ = os.Remove(logPath)
			}
		}()
	}

	file, err := os.Create(s.c.LogFile)
	if err != nil {
		return err
	}
	defer file.Close()
	return
}

func (s *systemv) configLogRoate() (err error) {
	if err = s.configLogFile(); err != nil {
		return err
	}
	if err = exec.Command("chown", "-R", s.c.User+":"+s.c.Group, path.Dir(s.c.LockFile)).Run(); err != nil {
		return err
	}
	tpl, err := template.New("logRoateConf").Parse(logRoateConf)
	if err != nil {
		return err
	}

	logrotateConfPath := "/etc/logrotate.d/" + s.c.Name
	file, err := os.OpenFile(logrotateConfPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(logrotateConfPath)
		}
	}()
	defer file.Close()

	if err = tpl.Execute(file, s.c); err != nil {
		return err
	}
	return
}

func (s *systemv) removeLogRoateConf() error {
	return os.Remove("/etc/logrotate.d/" + s.c.Name)
}

var logRoateConf = `/var/log/{{.Name}}/{{.Name}}.log {
	copytruncate
    daily
    rotate 10
    missingok
    notifempty
    dateext
	dateformat %Y%m%d
    compress
    delaycompress
	notifempty
	nomail
	noolddir
}`

var systemvScript = `#! /bin/sh
#
#       /etc/rc.d/init.d/{{.Name}}
#
#       Starts {{.Name}} as a daemon
#
# chkconfig: 2345 87 17
# description: {{.Description}}

{{$deps := "$network $time $named $local_fs"}}
{{- if .Dependencies}}
{{$deps = .Dependencies}}
{{- end}}
### BEGIN INIT INFO
# Provides: {{.Name}}
# Required-Start: {{$deps}}
# Required-Stop: {{$deps}}
# Default-Start: 2 3 4 5
# Default-Stop: 0 1 6
# Short-Description: start and stop {{.Name}}.
# Description: {{.Description}}
### END INIT INFO

#
# Source function library.
#
if [ -f /etc/rc.d/init.d/functions ]; then
    . /etc/rc.d/init.d/functions
fi
exec="{{.Exec}}"
args="{{.Args}}"
servname="{{.Name}}"
user="{{.User}}"
group="{{.Group}}"
pidfile="{{.PidFile}}"
lockfile="{{.LockFile}}"
workingDirectory="{{.WorkDir}}"
logFile="{{.LogFile}}"
[ -d $(dirname $lockfile) ] || mkdir -p $(dirname $lockfile)
[ -e /etc/sysconfig/$servname ] && . /etc/sysconfig/$servname

execPrifx=""
userName=` + "`whoami`" + `
if [ $userName == "root" ]; then
    execPrifx="su -l $user -c "
elif [ $userName != $user ]; then
    echo "only run with user root or $user"
    exit 1
fi

start() {
    [ -x $exec ] || exit 5
    if [ -f $pidfile ]; then
        if ! [ -d "/proc/$(cat $pidfile)" ]; then
            rm $pidfile
            if [ -f $lockfile ]; then
                rm $lockfile
            fi
        fi
    fi
    if ! [ -f $pidfile ]; then
        printf "Starting $servname:\t"
		cd ${workingDirectory}
        $execPrifx $exec $args &>> $logFile &
        echo $! > $pidfile
        touch $lockfile
        chown -R $user:$group $(dirname $logFile)
        chown -R $user:$group $(dirname $pidfile)
        chown -R $user:$group $(dirname $lockfile)
        success
        echo
    else
        # failure
        echo
        printf "$pidfile still exists...\n"
        exit 7
    fi
}
stop() {
    echo -n $"Stopping $servname: "
    killproc $servname
    retval=$?
    echo
    [ $retval -eq 0 ] && rm -f $lockfile
    return $retval
}
restart() {
    stop
    start
}
rh_status() {
    status -p $pidfile $servname
}
rh_status_q() {
    rh_status >/dev/null 2>&1
}
case "$1" in
    start)
        rh_status_q && exit 0
        $1
        ;;
    stop)
        rh_status_q || exit 0
        $1
        ;;
    restart)
        $1
        ;;
    status)
        rh_status
        ;;
    *)
        echo $"Usage: $0 {start|stop|status|restart}"
        exit 2
esac
exit $?
`
