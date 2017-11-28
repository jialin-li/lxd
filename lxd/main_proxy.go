package main

import (
	"fmt"
	"io"
	"io/ioutil"	
	"net"
	"os"	
	"strings"
	"syscall"
	"strconv"

	"github.com/lxc/lxd/shared"
)

func cmdProxyDevStart(args *Args) error {
	err := run(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
	return nil
}

func run(args *Args) error {
	if (len(args.Params) != 6) {
		return fmt.Errorf("Invalid arguments")
	}

	// Get all our arguments
	listenPid := args.Params[1]
	listenAddr := args.Params[2]
	connectPid := args.Params[3]
	connectAddr := args.Params[4]
	fd, _ := strconv.Atoi(args.Params[5])


	// Check where we are in initialization
	if !shared.PathExists(fmt.Sprintf("/proc/self/fd/%d", fd)) {
		fmt.Printf("Listening on %s in %s, forwarding to %s from %s\n", listenAddr, listenPid, connectAddr, connectPid)
		fmt.Printf("Setting up the listener\n")

		file, err := setUpFile(listenAddr)
		if err != nil {
			return err
		}
		defer file.Close()

		listenerFd := file.Fd()
		if err != nil {
			return fmt.Errorf("failed to duplicate the listener fd: %v", err)
		}

		fmt.Printf("Re-executing ourselves\n")

		args.Params[5] = strconv.Itoa(int(listenerFd))
		err = syscall.Exec("/proc/self/exe", args.Params, []string{})
		if err != nil {
			return fmt.Errorf("failed to re-exec: %v", err)
		}
	}

	// Re-create listener from fd
	listenFile := os.NewFile(uintptr(fd), "listener")
	listener, err := net.FileListener(listenFile)
	if err != nil {
		return fmt.Errorf("failed to re-assemble listener: %v", err)
	}

	fmt.Printf("Starting to proxy\n")

	// write proxy pid to master file for lxd
	args.Params[5] =  strconv.Itoa(-1)
	fileEntry := fmt.Sprintf("%d:%s\n", os.Getpid(), strings.Join(args.Params[0:], " "))
	err = ioutil.WriteFile(shared.VarPath("devices","proxy", "proxies.info"), []byte(fileEntry), 0644)

	// begin proxying
	for {
		// Accept a new client
		srcConn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: Failed to accept new connection: %v\n", err)
			continue
		}

		// Connect to the target
		fields := strings.SplitN(connectAddr, ":", 2)
		dstConn, err := net.Dial("tcp", fields[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: Failed to connect to target: %v\n", err)
			srcConn.Close()
			continue
		}

		go io.Copy(srcConn, dstConn)
		go io.Copy(dstConn, srcConn)
	}

	return nil
}

func setUpFile (listenAddr string) (os.File, error) {
	fields := strings.SplitN(listenAddr, ":", 2)

	if (fields[0] == "unix") {
		return unixFile(fields[1])
	} else if (fields[0] == "tcp") {
		return tcpFile(fields[1])
	}
	return os.File{}, fmt.Errorf("cannot resolve file from network type: %v", fields[0])
}

func unixFile(path string) (os.File, error) {
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return os.File{}, fmt.Errorf("cannot resolve socket address: %v", err)
	}

	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return os.File{}, fmt.Errorf("cannot bind socket: %v", err)
	}

	file, err := listener.File()
	if err != nil {
		return os.File{}, fmt.Errorf("failed to extra fd from listener: %v", err)
	}

	return *file, err
}

func tcpFile (path string) (os.File, error) {
	addr, err := net.ResolveTCPAddr("tcp", path)
	if err != nil {
		return os.File{}, fmt.Errorf("cannot resolve socket address: %v", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return os.File{}, fmt.Errorf("cannot bind socket: %v", err)
	}

	file, err := listener.File()
	if err != nil {
		return os.File{}, fmt.Errorf("failed to extra fd from listener: %v", err)
	}

	return *file, err
}

