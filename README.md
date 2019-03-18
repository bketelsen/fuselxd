# fuselxd
A FUSE filesystem that reads from [lxd](https://linuxcontainers.org) filesystems.

## Building

```
go get -u github.com/bketelsen/fuselxd
```

## Usage

Assuming you have a container named `containername` with a user inside the container called `containeruser`:

```
mkdir mycontainer
fuselxd /home/you/mycontainer/ containername /home/containeruser/containerpath
cd /home/you/mycontainer
ls
> your container files will show up here!
```

## Pretty Pictures

![command](assets/command.png?raw=true "Command")
![mounted filesystem](assets/mountedfs.png?raw=true "Mounted FS")
![source filesystem](assets/container.png?raw=true "Source File System")