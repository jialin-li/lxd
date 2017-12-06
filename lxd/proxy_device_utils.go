package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/lxc/lxd/shared"	
)

type proxyProcInfo struct {
	listenPid		string
	connectPid		string
	connectAddr		string
	listenAddr		string
}

func createProxyDevInfoFile(containerName string, proxyDev string, proxyPid int) error {
	filePath := shared.VarPath("devices", containerName, proxyDev)
	f, err := os.Create(filePath)

	if err != nil {
		return err 
	}

	defer f.Close()

	info := fmt.Sprintf("%d", proxyPid)
	_, err = f.WriteString(info)

	return err
}

func setupProxyProcInfo(c container, device map[string]string) (*proxyProcInfo, error) {	
	pid := c.InitPID()
	containerPid := strconv.Itoa(int(pid))
	lxdPid := strconv.Itoa(os.Getpid())

	connectAddr := device["connect"]
	listenAddr := device["listen"]

	listenPid := "-1"
	connectPid := "-1"

	if (device["bind"] == "container") {
		listenPid = containerPid
		connectPid = lxdPid
	} else if (device["bind"] == "host") {
		listenPid = lxdPid
		connectPid = containerPid
	} else {
		return nil, fmt.Errorf("No indicated binding side")
	}

	p := &proxyProcInfo{
		listenPid:		listenPid,
		connectPid:		connectPid,
		connectAddr:	connectAddr,
		listenAddr:		listenAddr,
	}

	return p, nil
}

func killProxyProc(devPath string) error {
	contents, err := ioutil.ReadFile(devPath)
	if err != nil {
		return err
	}

	pid, _ := strconv.Atoi(string(contents))
	if err != nil {
		return err
	}
	
	syscall.Kill(pid, syscall.SIGINT)
	os.Remove(devPath)	
	
	return nil
}
