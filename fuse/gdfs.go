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
	id2container = make(map[string]Cont)

	// connect gdfs
	utils.SetupProxyFromEnv()
	handler, err = drive.NewHandler("client_id.json")
	if err != nil {
		log.Fatal(err)
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("google drive"),
		fuse.Subtype("gdfs"),
		fuse.LocalVolume(),
		fuse.VolumeName("Hello gdfs!"),
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

var id2container map[string]Cont   // fileId->dir container
var id2parentdir map[string]string // fileId->parentdir
var id2content map[string][]byte   // fileId->content

// Dir implements both Node and Handle for the root directory.
type Cont struct {
	dirDirs    []fuse.Dirent
	name2id    map[string]string // name->fileId+('0' for dir and '1' for file)
	hasUpdated bool              // true -> newest
}

type Dir struct {
	fileId string
}

func (d Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 0 // let it get dynamic id automatic
	a.Mode = os.ModeDir | 0775
	return nil
}

func (d Dir) GetDirAll() {
	fmt.Println("GetDirAll")
	_, ok := id2container[d.fileId]
	if !ok || !id2container[d.fileId].hasUpdated {
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
		id2container[d.fileId] = Cont{
			dirDirs:    dirDirs,
			name2id:    name2id,
			hasUpdated: true,
		}
	}
}

func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	fmt.Println("Lookup")
	d.GetDirAll()
	id, ok := id2container[d.fileId].name2id[name]
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

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("ReadDirAll")
	d.GetDirAll()
	return id2container[d.fileId].dirDirs, nil
}

func (d Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	fmt.Println("Mkdir")
	d.GetDirAll()
	name := req.Name
	_, ok := id2container[d.fileId].name2id[name]
	// check existance
	if ok {
		return nil, fuse.EEXIST
	}
	// gd
	newDir, err := handler.MkDirUnder(name, d.fileId)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	// local
	id2container[d.fileId].name2id[name] = newDir.Id + "0"
	tmp := id2container[d.fileId]
	tmp.dirDirs = append(tmp.dirDirs, fuse.Dirent{
		Inode: utils.Str2u64(newDir.Id),
		Type:  fuse.DT_Dir,
		Name:  name,
	})
	id2container[d.fileId] = tmp
	return Dir{
		fileId: newDir.Id,
	}, nil
}

func (d Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	fmt.Println("Remove")
	d.GetDirAll()
	name := req.Name
	id, ok := id2container[d.fileId].name2id[name]
	// check existance
	if !ok {
		return fuse.ENOENT
	}
	// gd
	handler.DeleteFile(id[:len(id)-1])
	// local
	delete(id2container[d.fileId].name2id, name)
	tmp := id2container[d.fileId]
	for i, dir := range tmp.dirDirs {
		if dir.Name == name {
			tmp.dirDirs = append(tmp.dirDirs[:i], tmp.dirDirs[i+1:]...)
			break
		}
	}
	id2container[d.fileId] = tmp
	return nil
}

// TODO: for dir rename create

// func (d Dir) Link(ctx context.Context, req *fuse.LinkRequest, old fs.Node) (fs.Node, error) {
// 	fmt.Println("Link")
// 	return nil, nil
// 	// d.GetDirAll()
// 	// name := req.NewName
// 	// oldId := old.fileId
// 	// _, ok := id2container[d.fileId].name2id[name]
// 	// // check existance
// 	// if ok {
// 	// 	return nil, fuse.EEXIST
// 	// }
// 	// // gd
// 	// handler.LinkFile(oldId, d.fileId)

// }

func (d Dir) Rename(ctx context.Context, req *fuse.RenameRequest, _newDir fs.Node) error {
	fmt.Println("Rename")
	newDir, ok := _newDir.(Dir)
	d.GetDirAll()
	// check existance
	_, ok = id2container[newDir.fileId].name2id[req.NewName]
	if ok {
		return fuse.EEXIST
	}
	id, ok := id2container[d.fileId].name2id[req.OldName]
	if !ok {
		return fuse.ENOENT
	}
	// gd
	handler.MoveFile(id, d.fileId, newDir.fileId)
	if req.OldName != req.NewName {
		handler.RenameFile(id, req.NewName)
	}
	// local
	// oldDir
	delete(id2container[d.fileId].name2id, req.OldName)
	tmp := id2container[d.fileId]
	var dirent fuse.Dirent
	for i, dir := range tmp.dirDirs {
		if dir.Name == req.OldName {
			dirent = tmp.dirDirs[i]
			tmp.dirDirs = append(tmp.dirDirs[:i], tmp.dirDirs[i+1:]...)
			break
		}
	}
	id2container[d.fileId] = tmp
	// newDir
	id2container[newDir.fileId].name2id[req.NewName] = id
	tmp = id2container[newDir.fileId]
	tmp.dirDirs = append(tmp.dirDirs, dirent)
	id2container[newDir.fileId] = tmp
	return nil
}

func (d Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	fmt.Println("Create")
	d.GetDirAll()
	return nil, nil, nil
}

// File implements both Node and Handle for the hello file.
type File struct {
	fileId string
}

//const greeting = "hello, world\n"

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	fmt.Println("Attr")
	file, err := handler.GetFile(f.fileId)
	if err != nil {
		log.Fatal(err)
		return err
	}
	a.Inode = 0 // let it get dynamic id automatic, WARNING
	a.Mode = 0775
	a.Size = uint64(file.Size)
	fmt.Println(file.Size)
	//a.Size = uint64(len(greeting))
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	fmt.Println("ReadAll")
	content, ok := id2content[f.fileId]
	if ok {
		return content, nil
	}
	r, err := handler.OpenReader(f.fileId)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	content, err = ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	fmt.Println(len(content))
	//return []byte(greeting), nil
	return content, nil
}

// TODO: for file write

func (f File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	fmt.Println("Write")
	content, err := f.ReadAll(ctx)
	if err != nil {
		log.Fatal(err)
		return err
	}
	newLen := req.Offset + int64(len(req.Data))
	if newLen := int(newLen); newLen > len(content) {
		content = append(content, make([]byte, newLen-len(content))...)
	}
	resp.Size = copy(content[req.Offset:], req.Data)
	return nil
}
