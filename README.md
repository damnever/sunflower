## The most easy way to export your local port.

http://sunflower.damnever.com is available for testing!

NOTE:

- There is currently no plan to support UDP.
- You can use TCP to support the high level protocols those built on top of TCP, HTTP/1.x is a special case.
- Only one connection per tunnel, don't worry, it works fine..
- There are too many TODOs on front-end and..
- Server side cross platform build is not working, also build it on Windows may have problems..
- Anyway, get started now, the other things is not important..

### Quick Start

```
$ go get github.com/damnever/sunflower/cmds/sun/...
$ cd `go list -e -f '{{.Dir}}' github.com/damnever/sunflower`
$ sun -b -c etc/sun.server.yaml
# I assume it has opened a new tab on your browser:
#  1) Login
#  2) Create an agent
#  3) Download the agent
#  4) Run the agent
#  5) Create tunnels
# Add the following lines to your /etc/hosts if you want to use the subdomain feature
#  127.0.0.1 sunflower.test
#  127.0.0.1 <subdomain>.<username>.sunflower.test
#
# That's it! Your can deploy it to your own public server:
#  1) Build
#  2) Edit etc/sun.server.yaml, just remember the control panel address
#  3) Copy the bianry file and config file to where you want
#  4) Run it..
```

### LICENSE

[The BSD 3-Clause License](https://github.com/damnever/sunflower/blob/master/LICENSE)
