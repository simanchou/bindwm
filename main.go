package main

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05"

type DomainResponseData struct {
	IsPost       bool
	HaveError    bool
	ErrorMessage string
	Data         []Domain
}
type Todo struct {
	Name     string
	Position string
	Age      int
	Salary   string
}

type TodoPageData struct {
	IsPost       bool
	HaveError    bool
	ErrorMessage string
	Todos        []Todo
}

type Domain struct {
	Name    string
	Serial  int64
	Records []RecordINFO
	Created string
}

type RecordINFO struct {
	ID       string
	Record   string
	Type     string
	TTL      int
	PointsTo string
}

func main() {
	/*
		StorageDB, err := bolt.Open("bindwm.db", 0600, nil)
		if err != nil {
			log.Fatalf("init db fail, error: %s", err)
		}
		defer StorageDB.Close()
	*/

	// static file, such as css,js,images
	staticFiles := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", staticFiles))

	http.HandleFunc("/domain", domainList)

	http.ListenAndServe(":9001", nil)
}

func domainList(w http.ResponseWriter, r *http.Request) {
	StorageDB, err := bolt.Open("bindwm.db", 0600, nil)
	if err != nil {
		log.Fatalf("init db fail, error: %s", err)
	}
	defer StorageDB.Close()

	drd := DomainResponseData{
		IsPost:       false,
		HaveError:    false,
		ErrorMessage: "no error",
	}

	err = StorageDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("domains"))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d := Domain{}
			json.Unmarshal(v, &d)
			drd.Data = append(drd.Data, d)
		}
		return nil
	})

	switch r.Method {
	case "GET":
		fmt.Println("this is a GET request", time.Now())
		fmt.Println(r.Method)

		tmpl := template.Must(template.ParseFiles("tmpl/domain-list.html"))
		tmpl.Execute(w, drd)
	case "POST":
		fmt.Println("this is a POST request")
		fmt.Println(r.Method)
		fmt.Println(r.Header)
		r.ParseForm()
		fmt.Println(r.Form)
		fmt.Println(r.PostForm)
		fmt.Println("this is form value from name input:", r.PostFormValue("domain-name"))

		p, _ := ioutil.ReadAll(r.Body)
		fmt.Printf("%s\n", p)

		postData := TodoPageData{}
		json.NewDecoder(r.Body).Decode(&postData)
		fmt.Printf("%#v\n", postData)
		http.Redirect(w, r, "/domain", http.StatusSeeOther)
	default:
		fmt.Println("unknown method")
	}

}
