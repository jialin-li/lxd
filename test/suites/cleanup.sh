#!/bin/bash

shopt -s expand_aliases
source ~/.bash_aliases

cleanup() {

  lxc stop proxyTest
  lxc delete proxyTest

}

cleanup
