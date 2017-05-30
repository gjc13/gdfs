package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gjc13/gdfs/drive"
	"github.com/gjc13/gdfs/utils"
)

func main() {
	utils.SetupProxyFromEnv()
	handler, _ := drive.NewHandler("client_id.json")
	files, _ := handler.List("root")
	for _, file := range files {
		fmt.Println(file.Id, file.Name)
	}
	root, err := handler.GetRoot()
	fmt.Println(err)
	reader, _ := os.Open("./main.go")
	uploadFile, _ := handler.WriteToDir("main.go", reader, root.Id)
	fmt.Println(uploadFile.Id)
	reader.Close()

	newdir, _ := handler.MkDirUnder("test_dir2", root.Id)
	fmt.Println(newdir.Id)

	handler.MoveFile(uploadFile.Id, root.Id, newdir.Id)
	handler.LinkFile(uploadFile.Id, root.Id)
	handler.RenameFile(uploadFile.Id, "test.go")
	r, _ := handler.OpenReader(uploadFile.Id)
	targetFile, _ := os.OpenFile("test.go", os.O_WRONLY|os.O_CREATE, 0660)
	io.Copy(targetFile, r)
	r.Close()
	targetFile.Close()
	handler.DeleteFile(uploadFile.Id)
}
