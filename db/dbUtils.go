package db

import (
	"database/sql"
	"log"
	"github.com/norhe/vault-transit-datakey-example/models"

	_ "github.com/go-sql-driver/mysql"
	"strings"
)

//var db *sql.DB
const APP_NAME = "my_app"
var dsn = "vault:vaultpw@tcp(127.0.0.1:3306)/" + APP_NAME
var db, err = sql.Open("mysql", dsn)

func init()  {
	/*dsn := "vault:vaultpw@tcp(127.0.0.1:3306)/" + APP_NAME
	db, err := sql.Open("mysql", dsn)


	if err != nil {
		panic(err)
	}*/

	//defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatalln(err)
	}

	create_tables(db)
}

func create_tables(db *sql.DB) {
	_, err := db.Exec("USE " + APP_NAME)
	if err != nil {
		log.Fatalln(err)
	}

	create_user_table :=
		"CREATE TABLE IF NOT EXISTS `user_data`(" +
			"`user_id` INT(11) NOT NULL AUTO_INCREMENT, " +
			"`user_name` VARCHAR(256) NOT NULL," +
			"`first_name` VARCHAR(256) NULL, " +
			"`last_name` VARCHAR(256) NULL, " +
			"`address` VARCHAR(256) NOT NULL, " +
			"`datakey` VARCHAR(256) DEFAULT NULL, " +
			"PRIMARY KEY (user_id) " +
			") engine=InnoDB;"

	log.Println("Creating User table (if not exist)")

	_, err = db.Exec(create_user_table)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Creating files table (if not exist)")

	create_files_table :=
		"CREATE TABLE IF NOT EXISTS `user_files`( " +
			"`file_id` INT(11) NOT NULL AUTO_INCREMENT, " +
			"`user_id` INT(11) NOT NULL, " +
			"`mime_type` VARCHAR(256) DEFAULT NULL, " +
			"`file_name` VARCHAR(256) DEFAULT NULL, " +
			"`file` LONGBLOB DEFAULT NULL, " +
			"PRIMARY KEY (file_id) " +
			") engine=InnoDB;"

	_, err = db.Exec(create_files_table)
	if err != nil {
		log.Fatalln(err)
	}
}

func GetUserId(username, firstname, lastname string) int64 {
	rows, err := db.Query("SELECT user_id FROM users WHERE user_name = ? AND first_name = ? AND last_name = ?", username, firstname, lastname)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	usr := models.User{}
	for rows.Next() {
		err := rows.Scan(&usr.ID)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return usr.ID
}

func GetFileFromDB(fname string, user_id int64) (models.UserFile, error) {
	var f models.UserFile
	f.FileName = fname
	f.UserID = user_id

	query := "SELECT file_id, `file`, mime_type FROM user_files WHERE file_name = ? AND user_id = ?"

	rows, err := db.Query(query, fname, user_id)

	if err != nil {
		log.Fatalln(err)
	}

	for rows.Next() {
		rows.Scan(&f.ID, &f.File, f.Mimetype)
	}

	return f, err
}

func GetUsers(limit int) []models.User {
	log.Println(db)
	rows, err := db.Query(
		`SELECT ud.user_id,
			ud.user_name,
			ud.first_name,
			ud.last_name,
			ud.address,
			GROUP_CONCAT(uf.file_name SEPARATOR ',')
	 FROM user_data AS ud, user_files AS uf
	 WHERE ud.user_id=uf.user_id
	 GROUP BY ud.user_id
	 LIMIT ?;`, limit)

	if err != nil {
		log.Println(err)
	}

	users := make([]models.User, 0, 10)
	for rows.Next() {
		usr := models.User{}
		var fnames string
		rows.Scan(&usr.ID, &usr.Username, &usr.FirstName, &usr.LastName, &usr.Address, &fnames)
		usr.FileNames = strings.Split(fnames, ",")
		users = append(users, usr)
	}
	log.Println(users)

	return users
}

func GetUserByName(username, firstname, lastname string) models.User {
	var usr models.User
	rows, err := db.Query(`SELECT user_id, user_name, first_name, last_name, address
	FROM users
	WHERE user_name = ?
	AND first_name = ?
	AND last_name = ?`,
		username, firstname, lastname)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		usr := models.User{}
		rows.Scan(&usr.ID, &usr.Username, &usr.FirstName, &usr.LastName, &usr.Address)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return usr
}

func GetUserByID(user_id int64) models.User {
	var usr models.User
	rows, err := db.Query(
		`SELECT ud.user_id,
			ud.user_name,
			ud.first_name,
			ud.last_name,
			ud.address,
			ud.datakey,
			GROUP_CONCAT(uf.file_name SEPARATOR ',')
	 FROM user_data AS ud, user_files AS uf
	 WHERE ud.user_id=?
	 AND ud.user_id=uf.user_id
	 GROUP BY ud.user_id`, user_id)

	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		var fnames string
		rows.Scan(&usr.ID, &usr.Username, &usr.FirstName, &usr.LastName, &usr.Address, &usr.Datakey, &fnames)
		usr.FileNames = strings.Split(fnames, ",")
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return usr
}

func CreateUserWithDatakey(usr models.User, ciphertext string) (sql.Result, error) {
	result, err := db.Exec(
		"INSERT INTO user_data (user_name, first_name, last_name, address, datakey) VALUES (?, ?, ?, ?, ?)",
		usr.Username,
		usr.FirstName,
		usr.LastName,
		usr.Address,
		ciphertext,
	)

	if err != nil {
		log.Println(err)
	}
	return result, err
}

func CreateUserFile(userId int64, mimeType, fileName string, encFile []byte) (sql.Result, error){
	result, err := db.Exec(
		"INSERT INTO `user_files` (`user_id`, `mime_type`, `file_name`, `file`) VALUES (?, ?, ?, ?)",
		userId,
		mimeType, // need user_id
		fileName,
		encFile,
	)
	if err != nil {
		log.Printf("Error saving file: %s", err)
	}

	return result, err
}