package main

import (
	"fmt"
	"io"
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
	if (len(args.Params) != 5) {
		return fmt.Errorf("Invalid number of arguments")
	}

	// Get all our arguments
	listenPid := args.Params[0]
	listenAddr := args.Params[1]
	connectPid := args.Params[2]
	connectAddr := args.Params[3]

	fd := -1
	if args.Params[4] != "0" {
		fd, _ = strconv.Atoi(args.Params[4])
	}	
	
	// Check where we are in initialization
	if !shared.PathExists(fmt.Sprintf("/proc/self/fd/%d", fd)) {
		fmt.Fprintf(os.Stdout, "Listening on %s in %s, forwarding to %s from %s\n", listenAddr, listenPid, connectAddr, connectPid)

		file, err := setUpFile(listenAddr)
		if err != nil {
			return err
		}
		defer file.Close()

		listenerFd := file.Fd()
		if err != nil {
			return fmt.Errorf("failed to duplicate the listener fd: %v", err)
		}

		newFd, _ := syscall.Dup(int(listenerFd))

		fmt.Fprintf(os.Stdout, "Re-executing ourselves\n")

		args.Params[4] = strconv.Itoa(int(newFd))
		execArgs := append([]string{"lxd" ,"proxydevstart"}, args.Params...)

		err = syscall.Exec("/proc/self/exe", execArgs, []string{})
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

	defer listener.Close()

	fmt.Fprintf(os.Stdout, "Starting to proxy\n")

	// begin proxying
	for {
		// Accept a new client
		srcConn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: Failed to accept new connection: %v\n", err)
			continue
		}
		fmt.Printf("Accepted a new connection\n")
		// b, err := ioutil.ReadAll(srcConn)
		// fmt.Printf("%s %d", b, len(b))
		// Connect to the target
		dstConn, err := getDestConn(connectAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: Failed to connect to target: %v\n", err)
			srcConn.Close()
			continue
		}
		fmt.Printf("Created dest connection and about to copy\n")

		go io.Copy(srcConn, dstConn)
		go io.Copy(dstConn, srcConn)
	}

	return nil
}

func getDestConn(connectAddr string) (net.Conn, error) {
	fields := strings.SplitN(connectAddr, ":", 2)
	if fields[0] == "tcp" {
		dstConn, err := net.Dial("tcp", strings.Join(fields[1:], ""))
		return dstConn, err
	} else if fields[0] == "unix" {
		dstConn, err := net.Dial("unix", strings.Join(fields[1:], ""))
		return dstConn, err
	} else {
		return nil, fmt.Errorf("Invalid connect addr type\n")
	}
}

func setUpFile(listenAddr string) (os.File, error) {
	fields := strings.SplitN(listenAddr, ":", 2)
	ipPortPair := strings.Join(fields[1:], "")

	if (fields[0] == "unix") {
		return unixFile(ipPortPair)
	} else if (fields[0] == "tcp") {
		return tcpFile(ipPortPair)
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

