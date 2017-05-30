package drive

import (
	"fmt"

	drive "google.golang.org/api/drive/v3"
)

func (handler *DriveHandler) GetRoot() (*drive.File, error) {
	return handler.srv.Files.Get("root").Fields("id, name").Do()
}

func (handler *DriveHandler) MkDirUnder(dirName string, parentDirId string) (*drive.File, error) {
	dirMeta := &drive.File{
		Name:     dirName,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentDirId},
	}
	return handler.srv.Files.Create(dirMeta).Fields("id, name").Do()
}

func (handler *DriveHandler) List(fileId string) ([]*drive.File, error) {
	pageToken := ""
	files := []*drive.File{}
	for {
		r, err := handler.srv.Files.List().
			Q(fmt.Sprintf("'%s' in parents", fileId)).PageToken(pageToken).
			PageSize(20).Fields("nextPageToken, files(id, name, mimeType)").Do()
		pageToken = r.NextPageToken
		files = append(files, r.Files...)
		if err != nil {
			return nil, err
		}
		if len(pageToken) == 0 {
			break
		}
	}
	return files, nil
}

func (handler *DriveHandler) MoveFile(fileId string, fromDirId string, toDirId string) error {
	if fromDirId == toDirId {
		return nil
	}
	file, err := handler.srv.Files.Get(fileId).Fields("name").Do()
	if err != nil {
		return err
	}
	_, err = handler.srv.Files.Update(fileId, file).
		AddParents(toDirId).RemoveParents(fromDirId).Do()
	return err
}

func (handler *DriveHandler) LinkFile(fileId string, toDirId string) error {
	file, err := handler.srv.Files.Get(fileId).Fields("name").Do()
	if err != nil {
		return err
	}
	_, err = handler.srv.Files.Update(fileId, file).AddParents(toDirId).Do()
	return err
}

func (handler *DriveHandler) DeleteFile(fileId string) error {
	return handler.srv.Files.Delete(fileId).Do()
}

func (handler *DriveHandler) RenameFile(fileId string, newName string) (*drive.File, error) {
	file, err := handler.srv.Files.Get(fileId).Fields("name").Do()
	if err != nil {
		return nil, err
	}
	file.Name = newName
	return handler.srv.Files.Update(fileId, file).Fields("id, name").Do()
}
