package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// newDNSMasqFile creates a new instance of a dnsNameFile
func newDNSMasqFile(domainName, networkInterface, networkName string, multiDomain bool) (dnsNameFile, error) {
	dnsMasqBinary, err := exec.LookPath("dnsmasq")
	if err != nil {
		return dnsNameFile{}, errors.Errorf("the dnsmasq cni plugin requires the dnsmasq binary be in PATH")
	}
	masqConf := dnsNameFile{
		ConfigFile:       makePath(networkName, confFileName),
		Domain:           domainName,
		PidFile:          makePath(networkName, pidFileName),
		NetworkInterface: networkInterface,
		AddOnHostsFile:   makePath(networkName, hostsFileName),
		Binary:           dnsMasqBinary,
	}
	if multiDomain {
		masqConf.LocalServersConfFile = makePath(networkName, localServersConfFileName)
		masqConf.OwnServersConfFile = makePath(networkName, ownServersConfFileName)
	}
	return masqConf, nil
}

// hup sends a sighup to a running dnsmasq to reload its hosts file. if
// there is no instance of the dnsmasq, then it simply starts it.
func (d dnsNameFile) hup() error {
	// First check for pidfile; if it does not exist, we just
	// start the service
	isRunning, pid := d.isRunning()
	if !isRunning {
		return d.start()
	}
	return pid.Signal(unix.SIGHUP)
}

// determines if selected dnsmasq instance is running
// it sends a signal 0 to the pid to determine if it
// responds or not
func (d dnsNameFile) isRunning() (bool, *os.Process) {
	if _, err := os.Stat(d.PidFile); os.IsNotExist(err) {
		return false, nil
	}
	pid, err := d.getProcess()
	if err != nil {
		return false, nil
	}
	if err := pid.Signal(syscall.Signal(0)); err != nil {
		return false, nil
	}
	return true, pid
}

// start starts the dnsmasq instance.
func (d dnsNameFile) start() error {
	args := []string{
		"-u",
		"root",
		fmt.Sprintf("--conf-file=%s", d.ConfigFile),
	}
	output, err := exec.Command(d.Binary, args...).CombinedOutput()
	if err != nil {
		return errors.Errorf("Message: %s, err: %v", string(output), err)
	}

	return nil
}

// stop stops the dnsmasq instance.
func (d dnsNameFile) stop() error {
	pid, err := d.getProcess()
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = pid.Kill(); err != nil {
		if strings.Contains(err.Error(), "process already finished") {
			return nil
		}
		return err
	}
	return nil
}

// getProcess reads the PID for the dnsmasq instance and returns an
// *os.Process. Returns an error if the PID does not exist.
func (d dnsNameFile) getProcess() (*os.Process, error) {
	pidFileContents, err := ioutil.ReadFile(d.PidFile)
	if err != nil {
		return nil, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidFileContents)))
	if err != nil {
		return nil, err
	}
	return os.FindProcess(pid)
}

// makePath formats a path name given a domain and suffix
func makePath(networkName, fileName string) string {
	// the generic path for where conf, host, pid files are kept is:
	// /run/containers/cni/dnsmasq/<network-name>/
	return filepath.Join(dnsNameConfPath(), networkName, fileName)
}
