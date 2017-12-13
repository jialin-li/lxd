#!/bin/bash

shopt -s expand_aliases
source ~/.bash_aliases

add_proxy_device() {
  lxc config device add $1 $2 proxy listen=tcp:127.0.0.1:$3 connect=tcp:127.0.0.1:$4 bind=$5
}

test_proxy_device() {
  
  lxc launch ubuntu:16.04 proxyTest
  add_proxy_device proxyTest test 1234 8888 host
  sleep 1
  lsc exec proxyTest -- nohup bash -c "nc -lkd 8888 > proxyTest.out &"
  exec 3>/dev/tcp/localhost/1234
  echo "Hello World, this your proxy device speaking!" >&3
  sleep 1
  #lsc exec proxyTest -- bash -c "exec 4</dev/tcp/localhost/8888"
  #lsc exec proxyTest -- bash -c "cat <&4"
}


test_proxy_device
