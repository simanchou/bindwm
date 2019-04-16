package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"html/template"
	"log"
	"net/http"
	"strconv"
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
	Records map[string]RecordINFO
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

	http.HandleFunc("/", domainList)
	http.HandleFunc("/domaindel", domainDel)
	http.HandleFunc("/record", recordList)
	http.HandleFunc("/recorddel", recordDel)

	http.ListenAndServe(":9001", nil)
}

func domainList(w http.ResponseWriter, r *http.Request) {
	StorageDB, err := bolt.Open("bindwm.db", 0600, nil)
	if err != nil {
		log.Fatalf("init db fail, error: %s", err)
	}
	defer StorageDB.Close()

	var domainsList []Domain

	err = StorageDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("domains"))
		if b == nil {
			return fmt.Errorf("first-time-running")
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d := Domain{}
			json.Unmarshal(v, &d)
			domainsList = append(domainsList, d)
		}
		return nil
	})

	if err != nil && strings.Contains(err.Error(), "first-time-running") {
		log.Printf("this is the first time running, you must add one domain to be continue")
	}

	switch r.Method {
	case "GET":
		log.Printf("%#v", domainsList)
		tmpl := template.Must(template.ParseFiles("tmpl/domain-list.html"))
		tmpl.Execute(w, domainsList)
	case "POST":
		r.ParseForm()
		domainForAdd := r.PostFormValue("domain-name")

		d := Domain{}
		err = StorageDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("domains"))
			if b == nil {
				return fmt.Errorf("first-time-running")
			}
			v := b.Get([]byte(domainForAdd))
			json.Unmarshal(v, &d)
			return nil
		})

		if d.Name == "" {
			log.Printf("begin to add domain %q to DB\n", domainForAdd)
			domain := &Domain{
				Name:    domainForAdd,
				Serial:  0,
				Records: make(map[string]RecordINFO),
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
			http.Redirect(w, r, "/", http.StatusSeeOther)
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
			http.Redirect(w, r, "/", http.StatusSeeOther)
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
		StorageDB, err := bolt.Open("bindwm.db", 0600, nil)
		if err != nil {
			log.Fatalf("init db fail, error: %s", err)
		}
		defer StorageDB.Close()

		r.ParseForm()
		domain := r.FormValue("domain")
		d1 := Domain{}
		err = StorageDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("domains"))
			v := b.Get([]byte(domain))
			json.Unmarshal(v, &d1)
			return nil
		})
		fmt.Printf("%#v\n", d1)

		tmpl := template.Must(template.ParseFiles("tmpl/record-list.html"))
		tmpl.Execute(w, d1)
	case "POST":
		StorageDB, err := bolt.Open("bindwm.db", 0600, nil)
		if err != nil {
			log.Fatalf("init db fail, error: %s", err)
		}
		defer StorageDB.Close()

		log.Println("this is a POST action")
		r.ParseForm()
		fmt.Println(r.PostForm)
		domain := r.PostFormValue("domain")
		record := r.PostFormValue("record")
		recordType := r.PostFormValue("record-type")
		pointsTo := r.PostFormValue("pointsto")
		ttl := r.PostFormValue("ttl")
		recordID := stringToMD5(record + recordType + pointsTo)
		fmt.Println(recordID, record, recordType, pointsTo, ttl)

		recordEntry := RecordINFO{
			ID:       recordID,
			Record:   record,
			Type:     recordType,
			PointsTo: pointsTo,
		}

		if len(ttl) > 0 {
			ttlInt, _ := strconv.Atoi(ttl)
			recordEntry.TTL = ttlInt
		} else {
			recordEntry.TTL = 600
		}

		d2 := Domain{}
		err = StorageDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("domains"))
			v := b.Get([]byte(domain))
			json.Unmarshal(v, &d2)
			return nil
		})
		fmt.Printf("%#v\n", d2)
		d2.Serial += 1
		d2.Records[recordID] = recordEntry

		err = StorageDB.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte("domains"))
			encoded, err := json.Marshal(d2)
			if err != nil {
				return nil
			}
			return b.Put([]byte(d2.Name), encoded)
		})

		fmt.Printf("%#v\n", d2)

		if err == nil {
			log.Printf("add record for domain %q successful\n", domain)
			http.Redirect(w, r, fmt.Sprintf("/record?domain=%s", domain), http.StatusSeeOther)
		} else {
			e := fmt.Sprintf("add record for domain  %q fail, error: %s\n", domain, err)
			w.Write([]byte(e))
		}

	default:
		log.Println("unknown action")
	}
}

func recordDel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		StorageDB, err := bolt.Open("bindwm.db", 0600, nil)
		if err != nil {
			log.Fatalf("init db fail, error: %s", err)
		}
		defer StorageDB.Close()

		fmt.Println("this is a GET action")
		r.ParseForm()
		fmt.Println(r.Form)
		domain := r.FormValue("domain")
		recordID := r.FormValue("record_id")

		d1 := Domain{}
		err = StorageDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("domains"))
			v := b.Get([]byte(domain))
			json.Unmarshal(v, &d1)
			return nil
		})

		type dataForResponse struct {
			Name   string
			Record RecordINFO
		}

		dfr := dataForResponse{}
		dfr.Name = d1.Name
		dfr.Record = d1.Records[recordID]

		fmt.Printf("%#v\n", dfr)

		tmpl := template.Must(template.ParseFiles("tmpl/record-del.html"))
		tmpl.Execute(w, dfr)
	case "POST":
		StorageDB, err := bolt.Open("bindwm.db", 0600, nil)
		if err != nil {
			log.Fatalf("init db fail, error: %s", err)
		}
		defer StorageDB.Close()

		fmt.Println("this is a POST action")
		r.ParseForm()
		domain := r.PostFormValue("record-del-domain-input")
		recordID := r.PostFormValue("record-del-id-input")

		d2 := Domain{}
		err = StorageDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("domains"))
			v := b.Get([]byte(domain))
			json.Unmarshal(v, &d2)
			return nil
		})
		delete(d2.Records, recordID)
		d2.Serial += 1
		err = StorageDB.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte("domains"))
			encoded, err := json.Marshal(d2)
			if err != nil {
				return nil
			}
			return b.Put([]byte(d2.Name), encoded)
		})

		if err == nil {
			log.Printf("delete record for domain %q successful\n", domain)
			http.Redirect(w, r, fmt.Sprintf("/record?domain=%s", domain), http.StatusSeeOther)
		} else {
			e := fmt.Sprintf("delete record for domain  %q fail, error: %s\n", domain, err)
			w.Write([]byte(e))
		}

	default:
		fmt.Println("unknown action")
	}
}

func stringToMD5(s string) string {
	has := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", has)
}
