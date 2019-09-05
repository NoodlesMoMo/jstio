#!/bin/bash


curl -s 'http://127.0.0.1:2379/v2/keys/registry/venus/planet/app1/development/upstream/1' -XPUT -d 'value=192.168.56.1:1234'

