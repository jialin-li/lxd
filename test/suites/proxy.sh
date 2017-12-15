#!/bin/bash

test_proxy_device() {
  
  MESSAGE="Proxy device test string"

  lxc launch ubuntu:16.04 proxyTester
  lxc config device add proxyTester proxyDev proxy listen=tcp:127.0.0.1:1234 connect=tcp:127.0.0.1:4321 bind=host
  if [[ $(lxc config device list proxyTester) != "proxyDev" ]]; then
    echo "Proxy device was not added to container"
  fi


  lxc config device remove proxyTester proxyDev
  if [[ $(lxc config device list proxyTester) ]]; then 
    echo "Proxy device was not removed from container"
  fi

  lxc config device add proxyTester proxyDev proxy listen=tcp:127.0.0.1:1234 connect=tcp:127.0.0.1:4321 bind=host
  lxc stop proxyTester
  if [[ $(lxc config device list proxyTester) != "proxyDev" ]]; then
    echo "Proxy device should not be deleted from config on container stop"
  fi
  lxc delete proxyTester

  lxc launch ubuntu:16.04 proxyTester
  if [[ $(lxc config device list proxyTester) ]]; then 
    echo "Proxy device was not deleted from config on container deletion"
  fi


  lxc config device add proxyTester proxyDev proxy listen=tcp:127.0.0.1:1234 connect=tcp:127.0.0.1:4321 bind=host
  sleep 1 

  lxc exec proxyTester -- nohup bash -c "nc -lkd 4321 > proxyTest.out &"
  exec 3>/dev/tcp/localhost/1234
  echo ${MESSAGE} >&3
  sleep 1
  if [[ $(lxc exec proxyTester -- bash -c "cat proxyTest.out") != ${MESSAGE} ]]; then
    echo "Proxy device did not properly send data from host to container"
  fi

  lxc config device remove proxyTester proxyDev
  echo ${MESSAGE} >&3
  sleep 1
  if [[ $(lxc exec proxyTester -- bash -c "cat proxyTest.out") != ${MESSAGE} ]]; then
    echo "Proxy device was not correctly stopped"
  fi

  lxc stop proxyTester
  lxc delete proxyTester

  lxc launch ubuntu:16.04 proxyTester
  lxc config device add proxyTester proxyDev proxy listen=tcp:127.0.0.1:1234 connect=tcp:127.0.0.1:4321 bind=container
  sleep 1
  nc -lkd 4321 > proxyTest.out &
  netcat=$!
  lxc exec proxyTester -- bash -c "echo ${MESSAGE} > /dev/tcp/localhost/1234"
  if [[ $(cat proxyTest.out) != ${MESSAGE} ]]; then
    echo "Proxy device did not properly send data from container to host"
  fi

  kill $netcat
  rm proxyTest.out

  lxc restart proxyTester
  sleep 1
  nc -lkd 4321 > proxyTest.out &
  netcat=$!
  lxc exec proxyTester -- bash -c "echo ${MESSAGE} > /dev/tcp/localhost/1234"
  if [[ $(cat proxyTest.out) != ${MESSAGE} ]]; then
    echo "Proxy device did not restart on container restart"
  fi
  
  kill $netcat
  rm proxyTest.out


  lxc stop proxyTester
  lxc delete proxyTester

}


test_proxy_device
