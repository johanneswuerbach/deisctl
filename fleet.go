package deisctl

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/machine"
	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/ssh"
)

// Flags used for Fleet API connectivity
var Flags = struct {
	Debug                 bool
	Verbosity             int
	Version               bool
	Endpoint              string
	EtcdKeyPrefix         string
	UseAPI                bool
	KnownHostsFile        string
	StrictHostKeyChecking bool
	Tunnel                string
}{}

// global API client used by commands
var	cAPI client.API
// used to cache MachineStates
var machineStates map[string]*machine.MachineState
var requestTimeout time.Duration = time.Duration(10) * time.Second

func getTunnelFlag() string {
	tun := Flags.Tunnel
	if tun != "" && !strings.Contains(tun, ":") {
		tun += ":22"
	}
	return tun
}

func getChecker() *ssh.HostKeyChecker {
	if !Flags.StrictHostKeyChecking {
		return nil
	}
	keyFile := ssh.NewHostKeyFile(Flags.KnownHostsFile)
	return ssh.NewHostKeyChecker(keyFile)
}

func getFakeClient() (*registry.FakeRegistry, error) {
	return registry.NewFakeRegistry(), nil
}

func getRegistryClient() (client.API, error) {
	var dial func(string, string) (net.Conn, error)
	tun := getTunnelFlag()
	if tun != "" {
		sshClient, err := ssh.NewSSHClient("core", tun, getChecker(), false)
		if err != nil {
			return nil, fmt.Errorf("failed initializing SSH client: %v", err)
		}
		dial = func(network, addr string) (net.Conn, error) {
			tcpaddr, err := net.ResolveTCPAddr(network, addr)
			if err != nil {
				return nil, err
			}
			return sshClient.DialTCP(network, nil, tcpaddr)
		}
	}
	trans := http.Transport{
		Dial: dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	return client.NewRegistryClient(&trans, Flags.Endpoint, Flags.EtcdKeyPrefix, requestTimeout)
}

// randomMachineID return a random machineID from the Fleet cluster
func randomMachineID(c *FleetClient) (machineID string, err error) {
	machineState, err := c.Fleet.Machines()
	if err != nil {
		return "", err
	}
	var machineIDs []string
	for _, ms := range machineState {
		machineIDs = append(machineIDs, ms.ID)
	}
	return randomValue(machineIDs), nil
}
