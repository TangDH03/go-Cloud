package mr

import "mime/multipart"

type UploadJob struct {
	UsrName string
	Md5sum  string
	File    *multipart.FileHeader
	Dir     string
}
type UploadBigJob struct {
	UsrName  string
	Md5sum   string
	File     *multipart.FileHeader
	Dir      string
	Fragment int
	Blocks   int
}
