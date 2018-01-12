package main

import (
  "html/template"
  "log"
  "net/http"
  "encoding/base64"
  "io/ioutil"
  "strconv"
  "strings"
  "io"
  "bytes"

  "github.com/norhe/vault-transit-datakey-example/models"
  "github.com/norhe/vault-transit-datakey-example/secure"
  "github.com/norhe/vault-transit-datakey-example/db"
)


var tpl *template.Template

func initTemplates() {
  tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
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
  f, err := db.GetFileFromDB(fname, user_id)
  if err != nil {
	log.Fatalf("Error retrieving file: %s", err)
  }

  usr := db.GetUserByID(user_id)
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

func createHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "create.gohtml", nil)
  if err != nil {
	log.Fatalln(err)
  }
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
  err := tpl.ExecuteTemplate(w, "view.gohtml", db.GetUsers(10))
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
	usr := db.GetUserByID(user_id)

	log.Printf("User: %+v", usr)

	err = tpl.ExecuteTemplate(w, "update.gohtml", usr)
	if err != nil {
	  log.Fatalln(err)
	}
  }
}

// TODO: Add me
func updateRecordHandler(w http.ResponseWriter, req *http.Request) {

	log.Println("handler")
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

	// encrypt PII with transit
	log.Printf("Sending %s to encrypt", usr.Address)
	enc_address, err := secure.EncryptString(usr.Address)

	usr.Address = string(enc_address)
	result, err := db.CreateUserWithDatakey(usr, ciphertext)

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

	  _, err = db.CreateUserFile(user_id, handler.Header.Get("Content-Type"), handler.Filename, encryptedFile)

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