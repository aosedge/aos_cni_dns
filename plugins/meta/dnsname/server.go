package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// adds remote servers to existing dnsmasq instance
func addRemoteServers(fileConfig string, remoteServers []string) error {
	curServerItems, err := readServerItems(fileConfig)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	remoteServersItems := remoteServersToServerItems(remoteServers)
	mergedServerItems, modified := mergeServerItems(curServerItems, remoteServersItems)

	if !modified {
		return nil
	}

	return writeServerItems(fileConfig, mergedServerItems)
}

// adds local servers to existing dnsmasq instances
func addLocalServers(conf dnsNameFile, servers []string) error {
	serverItems := serversToServerItems(conf.Domain, servers)
	// write own servers to file
	if err := writeServerItems(conf.OwnServersConfFile, serverItems); err != nil {
		return err
	}

	// walk through existing dnsmasq and add new local servers
	curServersItems, err := readServerItems(conf.LocalServersConfFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	curDir := filepath.Base(filepath.Dir(conf.LocalServersConfFile))

	items, err := ioutil.ReadDir(filepath.Join(dnsNameConfPath()))
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.IsDir() && item.Name() != curDir {
			instanceServers, err := addServersToInstance(item.Name(), conf.Domain, serverItems)
			if err != nil {
				return err
			}
			curServersItems, _ = mergeServerItems(curServersItems, instanceServers)
		}
	}
	curServersItems, _ = removeServerItems(curServersItems, serverItems)
	return writeServerItems(conf.LocalServersConfFile, curServersItems)
}

// removes local servers from existing dnsmasq instances
func removeLocalServers(conf dnsNameFile, servers []string) error {
	serverItems := serversToServerItems(conf.Domain, servers)
	// walk through existing dnsmasq and remove local servers
	curDir := filepath.Base(filepath.Dir(conf.LocalServersConfFile))
	items, err := ioutil.ReadDir(filepath.Join(dnsNameConfPath()))
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.IsDir() && item.Name() != curDir {
			if err := removeServersFromInstance(item.Name(), serverItems); err != nil {
				return err
			}
		}
	}
	return nil
}

// adds server items to specific dnsmasq instance
func addServersToInstance(networkName, domainName string, serverItems []string) ([]string, error) {
	// set multiDomain as true in newDNSMasqFile as this code is called only for multi domain
	conf, err := newDNSMasqFile("", "", networkName, true)
	if err != nil {
		return nil, err
	}
	ownServerItems, err := readServerItems(conf.OwnServersConfFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if isDomainInList(domainName, ownServerItems) {
		return nil, errors.Errorf("domain %s already exists", domainName)
	}
	curServerItems, err := readServerItems(conf.LocalServersConfFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	mergedServerItems, modified := mergeServerItems(curServerItems, serverItems)
	// if server items modified, write them to the file
	if modified {
		if err := writeServerItems(conf.LocalServersConfFile, mergedServerItems); err != nil {
			return nil, err
		}
		// if instance is running send hup signal to apply new configuration
		if isRunning, _ := conf.isRunning(); isRunning {
			if err := conf.stop(); err != nil {
				return nil, err
			}
			if err := conf.start(); err != nil {
				return nil, err
			}
		}
	}
	// returns instance local servers + own servers
	return append(curServerItems, ownServerItems...), nil
}

// checks if server items has the domain name
func isDomainInList(domainName string, serverItems []string) bool {
	for _, item := range serverItems {
		fields := strings.Split(item, "/")
		if len(fields) < 3 {
			continue
		}
		if domainName == fields[1] {
			return true
		}
	}
	return false
}

// removes server items from specific dnsmasq instance
func removeServersFromInstance(networkName string, serverItems []string) error {
	// set multiDomain as true in newDNSMasqFile as this code is called only for multi domain
	conf, err := newDNSMasqFile("", "", networkName, true)
	if err != nil {
		return err
	}
	curServerItems, err := readServerItems(conf.LocalServersConfFile)
	if err != nil {
		return err
	}
	newServerItems, modified := removeServerItems(curServerItems, serverItems)
	// if server items modified, write them to the file
	if modified {
		if err := writeServerItems(conf.LocalServersConfFile, newServerItems); err != nil {
			return err
		}
		// if instance is running send hup signal to apply new configuration
		if isRunning, _ := conf.isRunning(); isRunning {
			if err := conf.stop(); err != nil {
				return err
			}
			if err := conf.start(); err != nil {
				return err
			}
		}
	}
	return nil
}

// merge two service slices. Return result slice and true if result slice was changed
func mergeServerItems(curServers []string, newServers []string) ([]string, bool) {
	modified := false
	for _, newServer := range newServers {
		found := false
		for _, curServer := range curServers {
			if newServer == curServer {
				found = true
				break
			}
		}
		if !found {
			curServers = append(curServers, newServer)
			modified = true
		}
	}
	return curServers, modified
}

// remove service slices from another slice. Return result slice and true if result slice was changed
func removeServerItems(curServers []string, removeServers []string) ([]string, bool) {
	modified := false
	for _, removeServer := range removeServers {
		for i, curServer := range curServers {
			if removeServer == curServer {
				curServers = append(curServers[:i], curServers[i+1:]...)
				modified = true
				break
			}
		}
	}
	return curServers, modified
}

// reads file to servers slice
func readServerItems(fileName string) ([]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	servers := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		servers = append(servers, scanner.Text())
	}
	return servers, scanner.Err()
}

// writes servers slice to file
func writeServerItems(fileName string, servers []string) error {
	sort.Strings(servers)
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, server := range servers {
		fmt.Fprintln(writer, server)
	}
	return writer.Flush()
}

// converts server IPs to dnsmasq
// generate servers items in dnsmasq config format: server=ip
// if resolution by domain name is required the format should be: server=/domain/ip
func serversToServerItems(domainName string, servers []string) []string {
	serverItems := make([]string, 0, len(servers))
	for _, server := range servers {
		serverItems = append(serverItems, fmt.Sprintf("server=/%s/%s", domainName, server))
	}
	return serverItems
}

func remoteServersToServerItems(remoteServer []string) []string {
	serverItems := make([]string, len(remoteServer))
	for i, server := range remoteServer {
		serverItems[i] = fmt.Sprintf("server=%s", server)
	}
	return serverItems
}
