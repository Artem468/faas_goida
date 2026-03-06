package file

type File struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	S3URL     string `json:"s3_url"`
	S3Key     string `json:"-"`
	ProjectID int64  `json:"project_id"`
}
