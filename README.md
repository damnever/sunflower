## The most easy way to export your local port.

**NOTE**:

- There is no plan to support UDP.
- It is intend for personal use, any other features beyond this may not be accepted.
- You can use TCP to support the high level protocols those built on top of TCP, HTTP/1.x is a special case.
- Only one connection per tunnel, don't worry, it works fine(I have been using it for months)..
- There are too many TODOs and..
- Server side cross platform build is not working, also build it on Windows may have problems..
- Code is dirty but works, anyway, get started now, the other things is not important..

### Quick Start

![sunflower.gif](./sunflower.gif)

```
$ go get github.com/damnever/sunflower/cmd/sun/...
$ cd `go list -e -f '{{.Dir}}' github.com/damnever/sunflower`
$ sun -b -c etc/sun.server.yaml
```

Your can deploy it to your own public server:
 1. Build.
 2. Edit etc/sun.server.yaml, just remember the control panel address.
 3. Copy the bianry file and config file to where you want.
 4. Using systemd or supervisord to manage the daemon process.

### LICENSE

[The BSD 3-Clause License](https://github.com/damnever/sunflower/blob/master/LICENSE)
