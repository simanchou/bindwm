package main

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05"

type DomainResponseData struct {
	IsPost       bool
	HaveError    bool
	ErrorMessage string
	Data         []Domain
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
	http.HandleFunc("/domaindel", domainDel)
	http.HandleFunc("/record", recordList)

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
		if b == nil {
			return fmt.Errorf("first-time-running")
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d := Domain{}
			json.Unmarshal(v, &d)
			drd.Data = append(drd.Data, d)
		}
		return nil
	})

	if err != nil && strings.Contains(err.Error(), "first-time-running") {
		log.Printf("this is the first time running, you must add one domain to be continue")
	}

	switch r.Method {
	case "GET":
		tmpl := template.Must(template.ParseFiles("tmpl/domain-list.html"))
		tmpl.Execute(w, drd)
	case "POST":
		r.ParseForm()
		domainForAdd := r.PostFormValue("domain-name")

		d := Domain{}
		err = StorageDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("domains"))
			v := b.Get([]byte(domainForAdd))
			json.Unmarshal(v, &d)
			return nil
		})

		if d.Name == "" {
			log.Printf("begin to add domain %q to DB\n", domainForAdd)
			domain := &Domain{
				Name:    domainForAdd,
				Serial:  0,
				Records: []RecordINFO{},
				Created: time.Now().Format(TimeFormat),
			}

			err = StorageDB.Update(func(tx *bolt.Tx) error {
				b, err := tx.CreateBucketIfNotExists([]byte("domains"))
				if err != nil {
					return err
				}
				encoded, err := json.Marshal(domain)
				if err != nil {
					return nil
				}

				return b.Put([]byte(domain.Name), encoded)
			})

			if err == nil {
				log.Printf("domain %q add to DB successful\n", domain.Name)
			}
			http.Redirect(w, r, "/domain", http.StatusSeeOther)
		} else {
			log.Printf("domain %q is exist\n", domainForAdd)
			d := Domain{Name: domainForAdd}
			tmpl := template.Must(template.ParseFiles("tmpl/error-domainisexist.html"))
			tmpl.Execute(w, d)
		}
	default:
		fmt.Println("unknown method")
	}
}

func domainDel(w http.ResponseWriter, r *http.Request) {
	StorageDB, err := bolt.Open("bindwm.db", 0600, nil)
	if err != nil {
		log.Fatalf("init db fail, error: %s", err)
	}
	defer StorageDB.Close()

	switch r.Method {
	case "GET":
		r.ParseForm()
		fmt.Println(r.Form)
		domainForDel := r.Form["domain"][0]
		log.Printf("begin to delete domain %q\n", domainForDel)

		domain := Domain{}
		err = StorageDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("domains"))
			v := b.Get([]byte(domainForDel))
			json.Unmarshal(v, &domain)
			return nil
		})

		tmpl := template.Must(template.ParseFiles("tmpl/domain-del.html"))
		tmpl.Execute(w, domain)
	case "POST":
		r.ParseForm()
		domainForDel := r.PostFormValue("domaindel-input")

		err = StorageDB.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("domains"))
			e := b.Delete([]byte(domainForDel))
			if e != nil {
				return e
			}
			return nil
		})

		if err == nil {
			log.Printf("delete domain %q successful\n", domainForDel)
			http.Redirect(w, r, "/domain", http.StatusSeeOther)
		} else {
			e := fmt.Sprintf("delete domain %q fail, error: %s\n", domainForDel, err)
			w.Write([]byte(e))
		}
	default:
		log.Println("unknown action")
	}
}

func recordList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		tmpl := template.Must(template.ParseFiles("tmpl/record-list.html"))
		tmpl.Execute(w, "")
	case "POST":
		log.Println("this is a POST action")
	default:
		log.Println("unknown action")
	}
}
