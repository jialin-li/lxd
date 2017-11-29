package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
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

func appendProxyDevInfoFile(containerName string, proxyInfo string) error {
	proxyDevFileLock.Lock()
	defer proxyDevFileLock.Unlock()

	filePath := shared.VarPath("networks", "proxy", containerName)
	f, err := os.OpenFile(filePath, os.O_APPEND | os.O_WRONLY | os.O_CREATE, 0644)
	if err != nil {
		// couldn't open/create the file 
		return err 
	}

	defer f.Close()
	
	_, err = f.Write([]byte(proxyInfo))
	if err != nil {
		return err
	}

	return nil
} 

func removeProxyDevInfoEntry(containerName string, proxyInfo string) error {
	proxyDevFileLock.Lock()
	defer proxyDevFileLock.Unlock()

	proxyDevFilePath := shared.VarPath("networks", "proxy", containerName)
	buf, err := ioutil.ReadFile(proxyDevFilePath)
	fileContents := string(buf)

	newContents := strings.Replace(fileContents, proxyInfo, "", -1)
	err = ioutil.WriteFile(proxyDevFilePath, []byte(newContents), 0644)
	return err
}

func restartProxyDev(d *Daemon, containerName string, proxyInfo string) error {
	// get name of target proxy device
	fields := strings.Split(proxyInfo, ":")
	targetDev := fields[1]

	// get the container by the container name
	c, err := containerLoadByName(d.State(), containerName)

	if !c.IsRunning() {
		return fmt.Errorf("Cannot restart proxy device for a stopped container")
	}

	// get the list of devices and check if it contains our target dev
	allDevices := c.LocalDevices() 
	if !allDevices.ContainsName(targetDev) {
		return fmt.Errorf("Exited proxy device could not be found to restart")
	}
	
	proxyValues, err := setupProxyProcInfo(c, allDevices[targetDev])

	// run the command, right now haven't figured out how to 
	proxyPid, _, err := shared.RunCommandGetPid(
					c.DaemonState().OS.ExecPath,
					"proxydevstart",
					proxyValues.listenPid,
					proxyValues.listenAddr,
					proxyValues.connectPid,
					proxyValues.connectAddr,
					"-1")	
	if err != nil {
		return err 
	}

	newProxyInfoPair := fmt.Sprintf("%d:%s\n", proxyPid, targetDev)

	err = appendProxyDevInfoFile(containerName, newProxyInfoPair)
	if err != nil {
		return err
	}

	return nil
}

func setupProxyProcInfo(c container, device map[string]string) (*proxyProcInfo, error) {	
	pid := c.InitPID()
	if pid == -1 {
		return nil, fmt.Errorf("Cannot add proxy device to stopped container")
	}

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
	proxyDevInfoPath := shared.VarPath("networks", "proxy", containerName)	
	buf, err := ioutil.ReadFile(proxyDevInfoPath)	

	if err != nil {
		return err
	}

	contents := string(buf)
	err = os.Remove(proxyDevInfoPath)	

	for _, proxyInfo := range strings.Split(contents, "\n") {
		fields := strings.Split(proxyInfo, ":")
		proxyPid, _ := strconv.Atoi(fields[0]) 
		p, _ := os.FindProcess(proxyPid)
		err = p.Kill()
	}	
	
	return nil
}

func killProxyProc(containerName string, devName string) error {
	proxyDevInfoPath := shared.VarPath("networks", "proxy", containerName)	
	buf, err := ioutil.ReadFile(proxyDevInfoPath)

	if err != nil {
		return err
	}

	contents := string(buf)

	for _, proxyInfo := range strings.Split(contents, "\n") {
		fields := strings.Split(proxyInfo, ":")
		if fields[1] == devName {
			proxyPid, _ := strconv.Atoi(fields[0]) 
			p, _ := os.FindProcess(proxyPid)
			err = p.Kill()
			return err
		}
	}
	
	return fmt.Errorf("No proxy process running for given device name")
}