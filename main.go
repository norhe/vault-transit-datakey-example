package main

import (
  "database/sql"
  "html/template"
  "log"
  "net/http"
  "encoding/base64"
  "io/ioutil"
  "strconv"
  "strings"
  "io"
  "bytes"

  _ "github.com/go-sql-driver/mysql"
  "github.com/hashicorp/vault/api"
  "github.com/norhe/vault-transit-datakey-example/models"
  "github.com/norhe/vault-transit-datakey-example/secure"
)

var db *sql.DB
var tpl *template.Template
var vlt *api.Client

const KEY_NAME = "my_app_key"

func initDB() *sql.DB {
  dsn := "vault:vaultpw@tcp(127.0.0.1:3306)/my_app"
  db, err := sql.Open("mysql", dsn)

  if err != nil {
	panic(err)
  }

  err = db.Ping()
  if err != nil {
	log.Fatalln(err)
  }

  create_tables(db)

  return db
}

func create_tables(db *sql.DB) {
  _, err := db.Exec("USE my_app")
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

func initTemplates() {
  tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
  db = initDB()
  defer db.Close()
  initTemplates()

  url := "0.0.0.0:1234" // Listen on all interfaces

  // set up routes
  http.Handle("/favicon.ico", http.NotFoundHandler())
  http.HandleFunc("/", indexHandler)
  http.HandleFunc("/view", viewHandler)
  http.HandleFunc("/create", createHandler)
  http.HandleFunc("/update/", updateHandler)
  http.HandleFunc("/updateRecord", updateRecordHandler)
  http.HandleFunc("/createRecord", createRecordHandler)
  http.HandleFunc("/file/", downloadHandler)

  // run the server
  log.Printf("Server is running at http://%s", url)
  log.Fatalln(http.ListenAndServe(url, nil))
}

func indexHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "index.gohtml", nil)
  if err != nil {
	log.Fatalln(err)
  }
}

// expected URL is something like /file/23/the_file.jpg
func downloadHandler(w http.ResponseWriter, req *http.Request) {
  p := req.URL.Path[len("/file/"):]

  if p == "" {
	log.Println("no arg")
	http.Error(w, "Get `filename` not specified in the URL", 400)
	return
  }

  elements := strings.Split(p, "/")
  user_id, err := strconv.ParseInt(elements[0], 10, 64)
  if err != nil {
	log.Fatalf("Error retrieving user_id: %s", err)
  }

  fname := elements[1]

  if fname == "" || user_id == 0 {
	http.Error(w, "Get `filename` not specified in the URL", 400)
	return
  }
  // retrieve the file
  f, err := getFileFromDB(fname, user_id)
  if err != nil {
	log.Fatalf("Error retrieving file: %s", err)
  }

  usr := getUserByID(user_id)
  if err != nil {
	log.Fatalf("Could not retrieve User for datakey: %s", err)
  }

  f_size := strconv.Itoa(len(f.File))
  // decrypt files
  key, err := secure.DecryptString(usr.Datakey)
  if err != nil {
	log.Fatalf("Could not decrypt datakey: %s", err)
  }

  log.Println(key)

  dec_file, err := secure.DecryptFile(f.File, key)

  if err != nil {
	log.Fatalf("Could not decrypt datakey: %s", err)
  }

  // send file to client
  // set headers so the browser knows it is a download
  w.Header().Set("Content-Disposition", "attachment; filename="+fname )
  w.Header().Set("Content-Type", f.Mimetype)
  w.Header().Set("Content-Length", f_size)

  //stream the file to the client
	io.Copy(w, bytes.NewReader(dec_file))
}

func getFileFromDB(fname string, user_id int64) (models.UserFile, error) {
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

func createHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "create.gohtml", nil)
  if err != nil {
	log.Fatalln(err)
  }
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "view.gohtml", getUsers(10))
  if err != nil {
	log.Fatalln(err)
  }
}

func updateHandler(w http.ResponseWriter, req *http.Request) {
  p := req.URL.Path[len("/update"):]

  if p == "" || p == "/" {
	log.Println("no arg")
	err := tpl.ExecuteTemplate(w, "update.gohtml", nil)
	if err != nil {
	  log.Fatalln(err)
	}
  } else {
	user_id, err := strconv.ParseInt(req.URL.Path[len("/update/"):], 0, 16)
	if err != nil {
	  log.Fatalln(err)
	}
	log.Printf("user_id: %d", user_id)
	usr := getUserByID(user_id)

	log.Printf("User: %+v", usr)

	err = tpl.ExecuteTemplate(w, "update.gohtml", usr)
	if err != nil {
	  log.Fatalln(err)
	}
  }
}

func updateRecordHandler(w http.ResponseWriter, req *http.Request) {
  log.Println("handler")
}

func getUsers(limit int) []models.User {
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

func getUserByName(username, firstname, lastname string) models.User {
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

func getUserByID(user_id int64) models.User {
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

func getUserId(username, firstname, lastname string) int64 {
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

// form endpoint
func createRecordHandler(w http.ResponseWriter, req *http.Request) {
  if req.Method == http.MethodPost {
	var err error
	usr := models.User{}
		usr.Username = req.FormValue("username")
		usr.FirstName = req.FormValue("firstname")
		usr.LastName = req.FormValue("lastname")
	usr.Address = req.FormValue("address")

	secret, err := secure.GetDatakey()
	if err != nil {
	  log.Println(err)
	}

	// retrieve ciphertext to save, plaintext to encrypt files
	ciphertext := secret.Data["ciphertext"].(string)
	plaintext, err := base64.StdEncoding.DecodeString(secret.Data["plaintext"].(string))
	if err != nil {
	  log.Printf("Error decoding base64: %s", err)
	}

	log.Printf("Secret ciphertext: %s, plaintext: %s", ciphertext, plaintext)

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

	file, handler, err := req.FormFile("userfile")
	if err == nil && file != nil && handler != nil {
	  log.Printf("Found a formfile: %s with headers: %s", handler.Filename, handler.Header.Get("Content-Type"))
	  if err != nil {
		log.Println(err)
	  }

	  filedata, err := ioutil.ReadAll(file)

	  if err != nil {
		log.Printf("Error reading file data: %s", err)
	  }

	  user_id, err := result.LastInsertId()
	  if err != nil {
		log.Println(err)
	  }

	  encryptedFile := secure.EncryptFile(filedata, plaintext)

	  _, err = db.Exec(
		"INSERT INTO `user_files` (`user_id`, `mime_type`, `file_name`, `file`) VALUES (?, ?, ?, ?)",
		user_id,
		handler.Header.Get("Content-Type"), // need user_id
		handler.Filename,
		encryptedFile,
	  )
	  defer file.Close()
	  if err != nil {
		log.Printf("Error saving file: %s", err)
	  }
	} else {
	  log.Printf("Error retrieving file: %s", err)
	}

	log.Printf("Saved from form: Username: %s, FirstName: %s, LastName: %s, Address: %s", usr.Username, usr.FirstName, usr.LastName, usr.Address)
		err = tpl.ExecuteTemplate(w, "create.gohtml", map[string]interface{} {
	  "success": true,
	  "username": usr.Username,
	})

	if err != nil {
	  log.Println(err)
	}
		return
	}
	http.Error(w, "Method Not Supported", http.StatusMethodNotAllowed)
}