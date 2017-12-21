package main

import (
  "database/sql"
  "html/template"
  "log"
  "net/http"

  _ "github.com/go-sql-driver/mysql"

)

//var db *sql.DB
var tpl *template.Template


type user struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
  Address   string
  Files     []user_file
  Datakey   string
}

type user_file struct {
  ID       int64
  UserID   int64
  Mimetype string
  FileName string
  File     []byte
}

func initDB() {
  dsn := "vault:vaultpw@tcp(127.0.0.1:3306)/my_app"
  db, err := sql.Open("mysql", dsn)

  defer db.Close()

  if err != nil {
    panic(err)
  }

  err = db.Ping()
  if err != nil {
  	log.Fatalln(err)
  }

  create_tables(db)
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
    "`mime_type` VARCHAR(256) DEFAULT NULL, " +
    "`photo` BLOB DEFAULT NULL, " +
    "`datakey` VARCHAR(256) DEFAULT NULL, " +
    "PRIMARY KEY (user_id) " +
    ") engine=InnoDB;"

  log.Println("Creating user table (if not exist)")

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
    "`file` BLOB DEFAULT NULL, " +
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
  initDB()
  initTemplates()

  url := "0.0.0.0:1234" // Listen on all interfaces

  // set up routes
  http.Handle("/favicon.ico", http.NotFoundHandler())
  http.HandleFunc("/", indexHandler)
  http.HandleFunc("/view", viewHandler)
  http.HandleFunc("/create", createHandler)
  http.HandleFunc("/update", updateHandler)
  http.HandleFunc("/upload", uploadHandler)

  // run the server
  log.Printf("Server is running at http://%s", url)
  log.Fatalln(http.ListenAndServe(url, nil))
}

func indexHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "index.html", nil)
  if err != nil {
    log.Fatalln(err)
  }
}

func createHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "create.html", nil)
  if err != nil {
    log.Fatalln(err)
  }
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "view.html", nil)
  if err != nil {
    log.Fatalln(err)
  }
}

func updateHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "update.html", nil)
  if err != nil {
    log.Fatalln(err)
  }
}

// form endpoint
func uploadHandler(w http.ResponseWriter, req *http.Request) {
  if req.Method == http.MethodPost {
		usr := user{}
		usr.Username = req.FormValue("username")
		usr.FirstName = req.FormValue("firstname")
		usr.LastName = req.FormValue("lastname")
    usr.Address = req.FormValue("address")

		/*_, e = db.Exec(
			"INSERT INTO users (username, first_name, last_name, password) VALUES (?, ?, ?, ?)",
			usr.Username,
			usr.FirstName,
			usr.LastName,
			usr.Password,
		)
		checkErr(e)*/
    log.Printf("Retrieved from form: Username: %s, FirstName: %s, LastName: %s, Address: %s", usr.Username, usr.FirstName, usr.LastName, usr.Address)
		err := tpl.ExecuteTemplate(w, "create.html", map[string]interface{} {
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
