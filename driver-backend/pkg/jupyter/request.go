package jupyter

type createFileRequest struct {
	Path string `json:"path"`
}

func newCreateFileRequest(path string) *createFileRequest {
	return &createFileRequest{
		Path: path,
	}
}
