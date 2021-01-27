package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type testServerData struct {
	networkName  string
	localServers string
	ownServers   string
}

func cleanupAll() error {
	return os.RemoveAll(dnsNameConfPath())
}

func createNetwork(networkName string, localServers string, ownServers string) error {
	networkDir := filepath.Join(dnsNameConfPath(), networkName)
	if err := os.MkdirAll(networkDir, 0700); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(networkDir,
		localServersConfFileName), []byte(localServers), 0700); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(networkDir,
		ownServersConfFileName), []byte(ownServers), 0700); err != nil {
		return err
	}
	return nil
}

func TestAddLocalServers(t *testing.T) {
	if err := cleanupAll(); err != nil {
		t.Fatalf("Can't cleanup: %v", err)
	}
	localServers := `server=192.168.2.1
server=192.168.3.1
`
	ownServers := `server=192.168.1.1
`
	if err := createNetwork("net1", localServers, ownServers); err != nil {
		t.Fatalf("Can't create network: %v", err)
	}
	localServers = `server=192.168.1.1
server=192.168.3.1
`
	ownServers = `server=192.168.2.1
`
	if err := createNetwork("net2", localServers, ownServers); err != nil {
		t.Fatalf("Can't create network: %v", err)
	}
	localServers = `server=192.168.1.1
server=192.168.2.1
`
	ownServers = `server=192.168.3.1
`
	if err := createNetwork("net3", localServers, ownServers); err != nil {
		t.Fatalf("Can't create network: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dnsNameConfPath(), "net4"), 0700); err != nil {
		t.Fatalf("Can't create network dir: %v", err)
	}
	conf, err := newDNSMasqFile("", "", "net4")
	if err != nil {
		t.Fatalf("Can't create conf: %v", err)
	}
	if err := addLocalServers(conf, []string{"192.168.4.1"}); err != nil {
		t.Fatalf("Can't add local servers: %v", err)
	}
	testData := []testServerData{
		{
			networkName: "net1",
			localServers: `server=192.168.2.1
server=192.168.3.1
server=192.168.4.1
`,
			ownServers: `server=192.168.1.1
`,
		},
		{
			networkName: "net2",
			localServers: `server=192.168.1.1
server=192.168.3.1
server=192.168.4.1
`,
			ownServers: `server=192.168.2.1
`,
		},
		{
			networkName: "net3",
			localServers: `server=192.168.1.1
server=192.168.2.1
server=192.168.4.1
`,
			ownServers: `server=192.168.3.1
`,
		},
		{
			networkName: "net4",
			localServers: `server=192.168.1.1
server=192.168.2.1
server=192.168.3.1
`,
			ownServers: `server=192.168.4.1
`,
		},
	}
	for _, item := range testData {
		networkDir := filepath.Join(dnsNameConfPath(), item.networkName)
		data, err := ioutil.ReadFile(filepath.Join(networkDir, localServersConfFileName))
		if err != nil {
			t.Fatalf("Can't read file: %v", err)
		}
		if string(data) != item.localServers {
			t.Errorf("Wrong local servers, got: %v, want: %v", string(data), item.localServers)
		}
		data, err = ioutil.ReadFile(filepath.Join(networkDir, ownServersConfFileName))
		if err != nil {
			t.Fatalf("Can't read file: %v", err)
		}
		if string(data) != item.ownServers {
			t.Errorf("Wrong own servers, got: %v, want: %v", string(data), item.ownServers)
		}
	}
}

func TestRemoveLocalServers(t *testing.T) {
	if err := cleanupAll(); err != nil {
		t.Fatalf("Can't cleanup: %v", err)
	}
	localServers := `server=192.168.2.1
server=192.168.3.1
server=192.168.4.1
`
	ownServers := `server=192.168.1.1
`
	if err := createNetwork("net1", localServers, ownServers); err != nil {
		t.Fatalf("Can't create network: %v", err)
	}
	localServers = `server=192.168.1.1
server=192.168.3.1
server=192.168.4.1
`
	ownServers = `server=192.168.2.1
`
	if err := createNetwork("net2", localServers, ownServers); err != nil {
		t.Fatalf("Can't create network: %v", err)
	}
	localServers = `server=192.168.1.1
server=192.168.2.1
server=192.168.4.1
`
	ownServers = `server=192.168.3.1
`
	if err := createNetwork("net3", localServers, ownServers); err != nil {
		t.Fatalf("Can't create network: %v", err)
	}
	localServers = `server=192.168.1.1
server=192.168.2.1
server=192.168.3.1
`
	ownServers = `server=192.168.4.1
`
	if err := createNetwork("net4", localServers, ownServers); err != nil {
		t.Fatalf("Can't create network: %v", err)
	}
	conf, err := newDNSMasqFile("", "", "net4")
	if err != nil {
		t.Fatalf("Can't create conf: %v", err)
	}
	if err := removeLocalServers(conf, []string{"192.168.4.1"}); err != nil {
		t.Fatalf("Can't add local servers: %v", err)
	}
	testData := []testServerData{
		{
			networkName: "net1",
			localServers: `server=192.168.2.1
server=192.168.3.1
`,
			ownServers: `server=192.168.1.1
`,
		},
		{
			networkName: "net2",
			localServers: `server=192.168.1.1
server=192.168.3.1
`,
			ownServers: `server=192.168.2.1
`,
		},
		{
			networkName: "net3",
			localServers: `server=192.168.1.1
server=192.168.2.1
`,
			ownServers: `server=192.168.3.1
`,
		},
		{
			networkName: "net4",
			localServers: `server=192.168.1.1
server=192.168.2.1
server=192.168.3.1
`,
			ownServers: `server=192.168.4.1
`,
		},
	}
	for _, item := range testData {
		networkDir := filepath.Join(dnsNameConfPath(), item.networkName)
		data, err := ioutil.ReadFile(filepath.Join(networkDir, localServersConfFileName))
		if err != nil {
			t.Fatalf("Can't read file: %v", err)
		}
		if string(data) != item.localServers {
			t.Errorf("Wrong local servers, got: %v, want: %v", string(data), item.localServers)
		}
		data, err = ioutil.ReadFile(filepath.Join(networkDir, ownServersConfFileName))
		if err != nil {
			t.Fatalf("Can't read file: %v", err)
		}
		if string(data) != item.ownServers {
			t.Errorf("Wrong own servers, got: %v, want: %v", string(data), item.ownServers)
		}
	}
}
