package main

import (
	"fmt"
	"github.com/couchbaselabs/gocouchbase"
	"html/template"
	"net/http"
)

var bucket *gocouchbase.Bucket
var tmpls map[string]*template.Template

const (
	entriesPerPage = 30
)

func executeTemplate(w http.ResponseWriter, name string, data interface{}) {
	if tmpls == nil {
		tmpls = make(map[string]*template.Template)
	}
	if tmpls[name] == nil {
		tmpls[name] = template.Must(template.ParseFiles("tmpls/"+name, "tmpls/"+"layout.html"))
	}
	tmpls[name].ExecuteTemplate(w, "base", data)
}

type Beer struct {
	Id        string
	BreweryId string
	Name      string
}

type tdBeerIndex struct {
	Results []Beer
}

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	executeTemplate(w, "welcome.html", nil)
}

type beerByNameRow struct {
	Id  string
	Key string
}

type beerDoc struct {
	BreweryId string `json:"brewery_id"`
	Name      string `json:"name"`
}

func beerIndexHandler(w http.ResponseWriter, r *http.Request) {
	vq := gocouchbase.NewViewQuery("beer", "by_name").Limit(entriesPerPage).Stale(gocouchbase.Before)
	rows := bucket.ExecuteViewQuery(vq)
	row := beerByNameRow{}
	var beers []Beer
	for rows.Next(&row) {
		beer := beerDoc{}
		if _, _, err := bucket.Get(row.Id, &beer); err != nil {
			fmt.Printf("Get Error: %v\n", err)
			continue
		}

		beers = append(beers, Beer{
			Id:        row.Id,
			BreweryId: beer.BreweryId,
			Name:      beer.Name,
		})
	}
	if err := rows.Close(); err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}

	data := tdBeerIndex{
		Results: beers,
	}
	executeTemplate(w, "beer/index.html", data)
}

func main() {
	cluster, _ := gocouchbase.Connect("couchbase://192.168.7.26")
	bucket, _ = cluster.OpenBucket("beer-sample", "")

	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/beers", beerIndexHandler)

	http.Handle("/css/", http.FileServer(http.Dir("static/")))
	http.Handle("/js/", http.FileServer(http.Dir("static/")))

	fmt.Printf("Starting server on :9980\n")
	http.ListenAndServe(":9980", nil)
}
