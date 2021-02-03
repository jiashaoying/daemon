package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

var (
	// errUnsupportedSystem appears if try to use service on system which is not supported by this release
	errUnsupportedSystem = errors.New("unsupported system")

	// errRootPrivileges appears if run installation or deleting the service without root privileges
	errRootPrivileges = errors.New("you must have root user privileges. possibly using 'sudo' command should help")

	// ErrAlreadyInstalled appears if service already installed on the system
	errAlreadyInstalled = errors.New("service has already been installed")

	// errNotInstalled appears if try to delete service which was not been installed
	errNotInstalled = errors.New("service is not installed")

	// errAlreadyRunning appears if try to start already running service
	errAlreadyRunning = errors.New("service is already running")

	// errAlreadyStopped appears if try to stop already stopped service
	errAlreadyStopped = errors.New("service has already been stopped")

	// errMissExecValue appears if the Exec filed hasn't been specified in Config
	errMissExecValue = errors.New("you must specify the executable path")

	// errConfigIsNil appears if the Config is nil when call New method
	errConfigIsNil = errors.New("the config can't be nil")
)

const (
	defaultDescription string = "manage the %s daemon"
	defaultLogFile     string = "/var/log/%s/%s.log"
	defaultUser        string = "root"
	defaultGroup       string = "root"
	defaultPidFile     string = "/var/run/%s.pid"
	defaultLockFile    string = "/var/lock/subsys/%s.lock"
)

type Daemon interface {
	Install() error
	Enable() error
	Disable() error
	Remove() error
	Start() error
	Stop() error
	Status() error
	Log() error
}

type config struct {
	Description  string // description
	Name         string // daemon name
	Exec         string // executable file
	Args         string // command line argument
	WorkDir      string
	Dependencies string
	User         string
	Group        string
	LogFile      string
	PidFile      string
	LockFile     string
}

type Configurator interface {
	apply(c *config)
}

type Option func(c *config)

func (f Option) apply(c *config) {
	f(c)
}
func WithDescription(des string) Configurator {
	return Option(func(c *config) {
		c.Description = des
	})
}
func WithName(name string) Configurator {
	return Option(func(c *config) {
		c.Name = name
	})
}
func WithExec(exec string) Configurator {
	return Option(func(c *config) {
		c.Exec = exec
	})
}
func WithArgs(args string) Configurator {
	return Option(func(c *config) {
		c.Args = args
	})
}
func WithWorkDir(workDir string) Configurator {
	return Option(func(c *config) {
		c.WorkDir = workDir
	})
}
func WithDependencies(deps string) Configurator {
	return Option(func(c *config) {
		c.Dependencies = deps
	})
}
func WithUser(user string) Configurator {
	return Option(func(c *config) {
		c.User = user
	})
}
func WithGroup(group string) Configurator {
	return Option(func(c *config) {
		c.Group = group
	})
}
func WithLogFile(logFile string) Configurator {
	return Option(func(c *config) {
		c.LogFile = logFile
	})
}
func WithPidFile(pidFile string) Configurator {
	return Option(func(c *config) {
		c.PidFile = pidFile
	})
}

func WithLockFile(lockFile string) Configurator {
	return Option(func(c *config) {
		c.LockFile = lockFile
	})
}

var selfWrapDaemon, _ = newDaemon(defaultConfig())

func Install() error {
	if selfWrapDaemon == nil {
		return errUnsupportedSystem
	}
	return selfWrapDaemon.Install()
}

func Enable() error {
	if selfWrapDaemon == nil {
		return errUnsupportedSystem
	}
	return selfWrapDaemon.Enable()
}

func Disable() error {
	if selfWrapDaemon == nil {
		return errUnsupportedSystem
	}
	return selfWrapDaemon.Disable()
}

func Remove() error {
	if selfWrapDaemon == nil {
		return errUnsupportedSystem
	}
	return selfWrapDaemon.Remove()
}

func Start() error {
	if selfWrapDaemon == nil {
		return errUnsupportedSystem
	}
	return selfWrapDaemon.Start()
}

func Stop() error {
	if selfWrapDaemon == nil {
		return errUnsupportedSystem
	}
	return selfWrapDaemon.Stop()
}

func Status() error {
	if selfWrapDaemon == nil {
		return errUnsupportedSystem
	}
	return selfWrapDaemon.Status()
}

func Log() error {
	if selfWrapDaemon == nil {
		return errUnsupportedSystem
	}
	return selfWrapDaemon.Log()
}

func New(options ...Configurator) (Daemon, error) {
	conf := defaultConfig()
	for _, op := range options {
		op.apply(conf)
	}
	if err := setupConfig(conf); err != nil {
		return nil, err
	}
	return newDaemon(conf)
}

func newDaemon(c *config) (d Daemon, err error) {
	initProgramName, err := check()
	if err != nil {
		return nil, err
	}
	switch initProgramName {
	case "supervisor":
		return &supervisord{c}, nil
	case "init":
		return &systemv{c}, nil
	case "systemd":
		return &systemd{c}, nil
	}
	return nil, errUnsupportedSystem
}

func setupConfig(conf *config) error {
	if conf == nil {
		return errConfigIsNil
	}
	if conf.Exec == "" {
		return errMissExecValue
	}
	if conf.Name == "" {
		conf.Name = path.Base(conf.Exec)
	}
	if conf.WorkDir == "" {
		conf.WorkDir = path.Dir(conf.Exec)
	}
	return nil
}

func defaultConfig() *config {
	p, err := filepath.Abs(os.Args[0])
	if err != nil {
		return nil
	}
	conf := &config{}
	conf.Exec = p
	conf.Name = path.Base(conf.Exec)
	conf.WorkDir = path.Dir(conf.Exec)
	conf.LogFile = fmt.Sprintf(defaultLogFile, conf.Name, conf.Name)
	conf.Description = fmt.Sprintf(defaultDescription, conf.Name)
	conf.User = defaultUser
	conf.Group = defaultGroup
	conf.PidFile = fmt.Sprintf(defaultPidFile, conf.Name)
	conf.LockFile = fmt.Sprintf(defaultLockFile, conf.Name)
	return conf
}

// check if support the os
// will return init program name when support and the err is nil
func check() (string, error) {
	out, err := exec.Command("ps", "-p1").CombinedOutput()
	if err != nil {
		return "", err
	}
	reg := regexp.MustCompile("(init|systemd)")
	sysManager := reg.FindString(string(out))
	if sysManager == "systemd" {
		return sysManager, nil
	}
	out, err = exec.Command("which", "supervisord").CombinedOutput()
	if err == nil {
		return "supervisor", nil
	}
	if sysManager == "" {
		return "", errUnsupportedSystem
	}
	return sysManager, nil
}

// Lookup path for executable file
func executablePath(name string) (string, error) {
	var lp string
	var err error
	if lp, err = exec.LookPath(name); err != nil {
		return "", err
	}
	return filepath.Abs(lp)
}

// Check root rights to use system service
func checkPrivileges() error {
	if output, err := exec.Command("id", "-g").Output(); err == nil {
		if gid, parseErr := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32); parseErr == nil {
			if gid == 0 {
				return nil
			}
			return errRootPrivileges
		}
	}
	return errUnsupportedSystem
}

func execCommandWithOutput(name string, arg ...string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func pathOrFileIsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
