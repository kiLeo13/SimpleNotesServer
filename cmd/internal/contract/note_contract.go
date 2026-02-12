package contract

const MaxNoteFileSizeBytes = 30 * 1024 * 1024

var ValidNoteFileTypes = []string{"pdf", "png", "jpg", "jpeg", "jfif", "webp", "gif", "mp4", "mp3"}

type NoteResponse struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Content     string   `json:"content,omitempty"`
	Tags        []string `json:"tags"`
	Visibility  string   `json:"visibility"`
	NoteType    string   `json:"note_type"`
	ContentSize int      `json:"content_size"`
	CreatedByID int64    `json:"created_by_id"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type NoteRequest struct {
	Name       string   `json:"name" validate:"required,min=2,max=80"`
	Visibility string   `json:"visibility" validate:"required,oneof=PUBLIC ORG PRIVATE"`
	Tags       []string `json:"tags" validate:"required,max=50,nodupes,dive,required,min=2,max=30,nospaces"`
}

type TextNoteRequest struct {
	Name       string   `json:"name" validate:"required,min=2,max=80"`
	Content    string   `json:"content" validate:"required,max=1000000"`
	NoteType   string   `json:"note_type" validate:"required,oneof=MARKDOWN FLOWCHART"`
	Visibility string   `json:"visibility" validate:"required,oneof=PUBLIC ORG PRIVATE"`
	Tags       []string `json:"tags" validate:"required,max=50,nodupes,dive,required,min=2,max=30,nospaces"`
}

type UpdateNoteRequest struct {
	Name       *string  `form:"name" validate:"omitempty,min=2,max=80"`
	Visibility *string  `form:"visibility" validate:"omitempty,oneof=PUBLIC ORG PRIVATE"`
	Tags       []string `form:"tags" validate:"omitempty,max=50,nodupes,dive,required,min=2,max=30,nospaces"`
}
