# arpingo

A tool that performs a ping-sweep of a network, and obtains results from the system's ARP cache. Only dynamic entries are displayed.

Note: echo-request messages are sent, but echo-reply responses are not monitored.

## Windows example

    > arp -a | grep 192 | grep dynamic
      192.168.0.1           d8-xx-xx-xx-xx-1d     dynamic
      192.168.0.19          d0-xx-xx-xx-xx-fd     dynamic
    > arp -d 192.168.0.19
    > arp -a | grep 192 | grep dynamic
      192.168.0.1           d8-xx-xx-xx-xx-1d     dynamic
    > go build .\arpingo.go
    > .\arpingo.exe 192.168.0.0/24
    d8:xx:xx:xx:xx:1d: 192.168.0.1
    d0:xx:xx:xx:xx:fd: 192.168.0.19
