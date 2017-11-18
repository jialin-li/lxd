package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/lxc/lxd/shared"
)

func cmdProxy(args *Args) error {
	if len(args.Params) != 5 {
		return fmt.Errorf("Invalid arguments")
	}

	// Get all our arguments
	listenPid := args.Params[1]
	listenAddr := args.Params[2]
	connectPid := args.Params[3]
	connectAddr := args.Params[4]

	// Check where we are in initialization
	if !shared.PathExists("/proc/self/fd/100") {
		fmt.Printf("Listening on %s in %s, forwarding to %s from %s\n", listenAddr, listenPid, connectAddr, connectPid)
		fmt.Printf("Setting up the listener\n")
		fields := strings.SplitN(listenAddr, ":", 2)

		addr, err := net.ResolveTCPAddr(fields[0], fields[1])
		if err != nil {
			return fmt.Errorf("failed to resolve listener address: %v", err)
		}

		listener, err := net.ListenTCP(fields[0], addr)
		if err != nil {
			return fmt.Errorf("failed to setup listener: %v", err)
		}

		file, err := listener.File()
		if err != nil {
			return fmt.Errorf("failed to extra fd from listener: %v", err)
		}
		defer file.Close()

		fd := file.Fd()
		err = syscall.Dup3(int(fd), 100, 0)
		if err != nil {
			return fmt.Errorf("failed to duplicate the listener fd: %v", err)
		}

		fmt.Printf("Re-executing ourselves\n")
		err = syscall.Exec("/proc/self/exe", os.Args, []string{})
		if err != nil {
			return fmt.Errorf("failed to re-exec: %v", err)
		}
	}

	// Re-create listener from fd
	listenFile := os.NewFile(100, "listener")
	listener, err := net.FileListener(listenFile)
	if err != nil {
		return fmt.Errorf("failed to re-assemble listener: %v", err)
	}

	fmt.Printf("Starting to proxy\n")
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