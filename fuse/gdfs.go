package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/gjc13/gdfs/drive"
	"github.com/gjc13/gdfs/utils"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"golang.org/x/net/context"
)

var handler *drive.DriveHandler

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	//flag.PrintDefaults()
}

func main() {
	// parse
	var err error
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	// init global
	bigMap = make(map[string]Cont)

	// connect gdfs
	utils.SetupProxyFromEnv()
	handler, err = drive.NewHandler("client_id.json")
	if err != nil {
		log.Fatal(err)
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("helloworld"),
		fuse.Subtype("hellofs"),
		fuse.LocalVolume(),
		fuse.VolumeName("Hello world!"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = fs.Serve(c, FS{rootDir: Dir{fileId: "root"}})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

// FS implements the hello world file system.
type FS struct {
	rootDir Dir
}

func (f FS) Root() (fs.Node, error) {
	return f.rootDir, nil
}

var bigMap map[string]Cont // fileId->dir container

// Dir implements both Node and Handle for the root directory.
type Cont struct {
	dirDirs    []fuse.Dirent
	name2id    map[string]string // name->fileId+('0' for dir and '1' for file)
	hasUpdated bool              // true -> newest
}

type Dir struct {
	fileId string
	// hasRead bool
	// name    string
	// cont *Cont
}

func (d Dir) GetDirAll() {
	_, ok := bigMap[d.fileId]
	if !ok || !bigMap[d.fileId].hasUpdated {
		files, err := handler.List(d.fileId)
		if err != nil {
			log.Fatal(err)
		}
		dirDirs := make([]fuse.Dirent, 0)
		name2id := make(map[string]string)
		for _, file := range files {
			var ftype fuse.DirentType
			if drive.IsMimeDir(file.MimeType) {
				ftype = fuse.DT_Dir
				name2id[file.Name] = file.Id + "0"
			} else {
				ftype = fuse.DT_File
				name2id[file.Name] = file.Id + "1"
			}
			dirDirs = append(dirDirs, fuse.Dirent{
				Inode: utils.Str2u64(file.Id),
				Type:  ftype,
				Name:  file.Name,
			})
		}
		bigMap[d.fileId] = Cont{
			dirDirs:    dirDirs,
			name2id:    name2id,
			hasUpdated: true,
		}
	}
}

func (d Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 0 // let it get dynamic id automatic
	a.Mode = os.ModeDir | 0775
	return nil
}

func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	d.GetDirAll()
	id, ok := bigMap[d.fileId].name2id[name]
	if ok {
		if id[len(id)-1] == '0' {
			// dir
			return Dir{
				fileId: id[:len(id)-1],
			}, nil
		} else {
			return File{
				fileId: id[:len(id)-1],
			}, nil
		}
	}
	return nil, fuse.ENOENT
}

// var dirDirs = []fuse.Dirent{
// 	{Inode: 2, Name: "hello", Type: fuse.DT_File},
// }

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.GetDirAll()
	return bigMap[d.fileId].dirDirs, nil
}

// File implements both Node and Handle for the hello file.
type File struct {
	fileId string
	// file   *googledrive.File
}

const greeting = "hello, world\n"

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	// TODO: use getfilesize
	// var err errord
	fileSize, err := handler.GetFileSize(f.fileId)
	if err != nil {
		log.Fatal(err)
		return err
	}
	a.Inode = 0 // let it get dynamic id automatic
	a.Mode = 0775
	fmt.Println(fileSize)
	a.Size = uint64(len(greeting))
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	r, err := handler.OpenReader(f.fileId)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	content, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	fmt.Println(len(content))
	return []byte(greeting), nil
	return content, nil
}
