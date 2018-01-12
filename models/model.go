package models

type UserFile struct {
	ID       int64
	UserID   int64
	Mimetype string
	FileName string
	File     []byte
}

type User struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	Address   string
	Files     []UserFile
	FileNames []string
	Datakey   string
}