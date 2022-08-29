#!/bin/sh

redsocks -c redsocks.conf

iptables -t nat -I PREROUTING -p tcp -d 192.168.100.1/24 -j REDIRECT --to-ports 1082