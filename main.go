package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	client "github.com/lxc/lxd/client"
	"golang.org/x/net/context"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT CONTAINER CONTAINERROOT\n", os.Args[0])
	flag.PrintDefaults()
}

var lxdclient *Client

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 3 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	container := flag.Arg(1)
	croot := flag.Arg(2)

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName(container),
		fuse.Subtype("lxdfusefs"),
		fuse.LocalVolume(),
		fuse.VolumeName(container),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	lxdclient, err = NewClient("/var/snap/lxd/common/lxd/unix.socket")

	if err != nil {
		log.Fatal(err)
	}
	fsys := NewFS(container, croot)
	err = fs.Serve(c, fsys)
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

type FS struct {
	root          Dir
	containerName string
	basePath      string
}

func NewFS(container, croot string) FS {
	return FS{
		root:          rootDir(container, croot),
		containerName: container,
		basePath:      croot,
	}
}

func (f FS) Root() (fs.Node, error) {
	return f.root, nil
}
func rootDir(containerName, croot string) Dir {
	_, response, err := lxdclient.conn.GetContainerFile("bazil", "/home/ubuntu/projects")
	if err != nil {
		panic(err)
	}

	return Dir{
		inode:         1,
		mode:          os.ModeDir,
		location:      containerName,
		name:          croot,
		fullPath:      croot,
		response:      response,
		containerName: containerName,
		basePath:      croot,
	}

}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	inode uint64
	//	client   *Client
	mode          os.FileMode
	location      string
	name          string
	fullPath      string
	containerName string
	basePath      string
	gid           uint64
	uid           uint64
	response      *client.ContainerFileResponse
}

func (d Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = d.inode
	a.Mode = d.mode
	a.Uid = uint32(d.response.UID)
	a.Gid = uint32(d.response.GID)
	return nil
}

func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {

	var de fs.Node
	for i, entry := range d.response.Entries {
		if entry == name {
			resp := d.getInfo(entry)
			inode := d.inode*100 + uint64(i) // FIXME, this limits
			switch resp.Type {
			case "directory":
				de = Dir{
					inode:         inode,
					mode:          os.ModeDir,
					location:      d.containerName,
					name:          entry,
					fullPath:      filepath.Join(d.fullPath, name),
					response:      resp,
					containerName: d.containerName,
					basePath:      d.basePath,
					gid:           uint64(resp.GID),
					uid:           uint64(resp.UID),
				}

			case "file":
				de = File{
					inode:         inode,
					mode:          os.FileMode(resp.Mode),
					location:      d.containerName,
					containerName: d.containerName,
					fullPath:      filepath.Join(d.fullPath, name),
					name:          entry,
					response:      resp,
					gid:           uint64(resp.GID),
					uid:           uint64(resp.UID),
				}
			}
			if de != nil {
				return de, nil
			}
		}
	}
	return nil, fuse.ENOENT
}

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var entries []fuse.Dirent
	for i, entry := range d.response.Entries {
		resp := d.getInfo(entry)
		var de fuse.Dirent
		inode := d.inode*100 + uint64(i) // FIXME, this limits
		switch resp.Type {
		case "directory":
			de = fuse.Dirent{Inode: inode, Name: entry, Type: fuse.DT_Dir}
		case "file":
			de = fuse.Dirent{Inode: inode, Name: entry, Type: fuse.DT_File}
		}
		entries = append(entries, de)
	}
	return entries, nil
}

func (d Dir) getInfo(name string) *client.ContainerFileResponse {
	fileName := filepath.Join(d.fullPath, name)

	_, response, err := lxdclient.conn.GetContainerFile("bazil", fileName)
	if err != nil {
		panic(err)
	}
	return response

}

func (d Dir) getFile(name string) File {

	fileName := filepath.Join(d.fullPath, name)

	_, response, err := lxdclient.conn.GetContainerFile("bazil", fileName)
	if err != nil {
		panic(err)
	}
	f := File{
		inode:         d.inode + 10, //FIXME
		mode:          os.FileMode(response.Mode),
		location:      d.containerName,
		containerName: d.containerName,
		fullPath:      filepath.Join(d.fullPath, name),
		name:          name,
		gid:           uint64(response.GID),
		uid:           uint64(response.UID),
		response:      response,
	}
	return f
}

// File implements both Node and Handle for the hello file.
type File struct {
	inode uint64
	//	client   *Client
	mode          os.FileMode
	location      string
	containerName string
	fullPath      string
	name          string
	gid           uint64
	uid           uint64
	response      *client.ContainerFileResponse
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = f.inode
	a.Mode = f.mode
	bb, err := f.ReadAll(context.Background())
	if err != nil {
		panic(err)
	}
	a.Size = uint64(len(bb))
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {

	body, _, err := lxdclient.conn.GetContainerFile(f.containerName, f.fullPath)
	if err != nil {
		panic(err)
	}
	bb, err := ioutil.ReadAll(body)
	return bb, nil
}
