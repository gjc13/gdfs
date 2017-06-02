package drive

import (
	"io"

	drive "google.golang.org/api/drive/v3"
)

func IsMimeDir(mime string) bool {
	return mime == "application/vnd.google-apps.folder"
}

func IsMimeGoogleDoc(mime string) bool {
	return mime == "application/vnd.google-apps.document" ||
		mime == "application/vnd.google-apps.presentation" ||
		mime == "application/vnd.google-apps.spreadsheet"
}

func (handler *DriveHandler) UpdateFile(fileId string, r io.Reader) (*drive.File, error) {
	file, err := handler.srv.Files.Get(fileId).Fields("name").Do()
	if err != nil {
		return nil, err
	}
	return handler.srv.Files.Update(fileId, file).Media(r).Fields("id, name").Do()
}

func (handler *DriveHandler) WriteToDir(filename string, r io.Reader, parentDirId string) (*drive.File, error) {
	fileMeta := &drive.File{
		Name:    filename,
		Parents: []string{parentDirId},
	}
	return handler.srv.Files.Create(fileMeta).Media(r).Fields("id, name").Do()
}

func (handler *DriveHandler) OpenReader(fileId string) (io.ReadCloser, error) {
	getter := handler.srv.Files.Get(fileId)
	file, err := getter.Do()
	if err != nil {
		return nil, err
	}
	if IsMimeDir(file.MimeType) {
		return nil, ERR_READDIR
	}
	if IsMimeGoogleDoc(file.MimeType) {
		resp, err := handler.srv.Files.Export(fileId, "application/pdf").Download()
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	} else {
		resp, err := handler.srv.Files.Get(fileId).Download()
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}
}
