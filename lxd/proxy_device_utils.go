package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"

	"github.com/lxc/lxd/shared"	
)

var proxyDevFileLock sync.Mutex

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

// for use when the user wants to delete a proxy device
func removeProxyDevInfoFile(containerName string, proxyDev string) error {
	proxyDevFilePath := shared.VarPath("devices", containerName, proxyDev)
	err := os.Remove(proxyDevFilePath)		

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

func killAllProxyProcs(containerName string) error {	
	proxyDevicesPath := shared.VarPath("devices", containerName)
	err := os.Chmod(proxyDevicesPath, 0400)

	files, err := ioutil.ReadDir(proxyDevicesPath)	

	if err != nil {
		return fmt.Errorf("Error reading directory of container proxy devices")
	}

	for _, proxyInfo := range files {
		devname := proxyInfo.Name()
		killProxyProc(containerName, devname)
	}

	os.Remove(proxyDevicesPath)

	return nil
}

func killProxyProc(containerName string, devName string) error {
	proxyDevFile := shared.VarPath("devices", containerName, devName)
	err := os.Chmod(proxyDevFile, 0400)

	contents, err := ioutil.ReadFile(proxyDevFile)
	if err != nil {
		return err
	}

	pid, _ := strconv.Atoi(string(contents))
	if err != nil {
		return err
	}

	process, _ := os.FindProcess(pid)
	if err != nil {
		return err
	}

	err = process.Kill()

	err = os.Remove(proxyDevFile)	
	
	return err
}
