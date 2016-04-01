package main

import (
	"encoding/json"
	"fmt"
	"github.com/couchbaselabs/gocb"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var bucket *gocb.Bucket
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
	Type      string `json:"type"`
	Id        string `json:"id,omitempty"`
	BreweryId string `json:"brewery_id"`
	Name      string `json:"name"`
}
type BeerFull struct {
	Type        string  `json:"type"`
	Id          string  `json:"id,omitempty"`
	BreweryId   string  `json:"brewery_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Style       string  `json:"style"`
	Category    string  `json:"category"`
	Abv         float64 `json:"abv"`
	Ibu         float64 `json:"ibu"`
	Srm         float64 `json:"srm"`
	Upc         float64 `json:"upc"`
}
type Brewery struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
type BreweryFull struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	City        string `json:"city"`
	State       string `json:"state"`
	Code        string `json:"code"`
	Country     string `json:"country"`
	Phone       string `json:"phone"`
	Website     string `json:"website"`
	Description string `json:"description"`
}

func parseFloat(val string) float64 {
	num, _ := strconv.ParseFloat(val, 64)
	return num
}

func beerFromForm(f url.Values) BeerFull {
	return BeerFull{
		BreweryId:   f.Get("beer_brewery_id"),
		Name:        f.Get("beer_name"),
		Description: f.Get("beer_description"),
		Style:       f.Get("beer_style"),
		Category:    f.Get("beer_category"),
		Abv:         parseFloat(f.Get("beer_abv")),
		Ibu:         parseFloat(f.Get("beer_ibu")),
		Srm:         parseFloat(f.Get("beer_srm")),
		Upc:         parseFloat(f.Get("beer_upc")),
	}
}

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	executeTemplate(w, "welcome.html", nil)
}

func removeHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.Split(r.URL.Path, "/")[3]
	_, err := bucket.Remove(id, 0)
	if err != nil {
		fmt.Printf("Remove Error: %v\n", err)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

type beerByNameRow struct {
	Id  string
	Key string
}

type tdBeerIndex struct {
	Results []Beer
}

func beerIndexHandler(w http.ResponseWriter, r *http.Request) {
	vq := gocb.NewViewQuery("beer", "by_name").Limit(entriesPerPage).Stale(gocb.Before)
	rows, err := bucket.ExecuteViewQuery(vq)
	if nil != err {
		panic(err)
	}
	var row beerByNameRow
	var beers []Beer
	for rows.Next(&row) {
		beer := Beer{}

		if _, err := bucket.Get(row.Id, &beer); err != nil {
			fmt.Printf("Get Error: %v\n", err)
			continue
		}

		beer.Id = row.Id
		beers = append(beers, beer)
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

func beerSearchHandler(w http.ResponseWriter, r *http.Request) {
	value := r.URL.Query().Get("value")

	vq := gocb.NewViewQuery("beer", "by_name").Limit(entriesPerPage).Stale(gocb.Before)
	vq.Range(value, value+"\u0FFFF", false)

	rows, err := bucket.ExecuteViewQuery(vq)
	if nil != err {
		panic(err)
	}

	var row beerByNameRow
	var beers []Beer
	for rows.Next(&row) {
		beer := Beer{}

		if _, err := bucket.Get(row.Id, &beer); err != nil {
			fmt.Printf("Get Error: %v\n", err)
			continue
		}

		beer.Id = row.Id
		beers = append(beers, beer)
	}
	if err := rows.Close(); err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}

	bytes, err := json.Marshal(beers)
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}

	w.Write(bytes)
}

type tdBeerShow struct {
	Beer       BeerFull
	BeerFields map[string]string
}

func beerShowHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.Split(r.URL.Path, "/")[3]

	var beer BeerFull
	if _, err := bucket.Get(id, &beer); err != nil {
		fmt.Fprintf(w, "Get Error: %v\n", err)
		return
	}

	beer.Id = id
	data := tdBeerShow{
		Beer: beer,
		BeerFields: map[string]string{
			"name":        beer.Name,
			"description": beer.Description,
			"style":       beer.Style,
			"category":    beer.Category,
			"abv":         strconv.FormatFloat(beer.Abv, 'f', -1, 64),
			"ibu":         strconv.FormatFloat(beer.Ibu, 'f', -1, 64),
			"srm":         strconv.FormatFloat(beer.Srm, 'f', -1, 64),
			"upc":         strconv.FormatFloat(beer.Upc, 'f', -1, 64),
		},
	}
	executeTemplate(w, "beer/show.html", data)
}

type tdBeerEdit struct {
	Beer     BeerFull
	IsCreate bool
}

func beerCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		data := tdBeerEdit{
			IsCreate: true,
		}
		executeTemplate(w, "beer/edit.html", data)
	} else {
		r.ParseForm()
		beer := beerFromForm(r.Form)
		beer.Type = "beer"

		id := strings.ToLower(beer.BreweryId + "-" + strings.Replace(beer.Name, " ", "-", 0))
		bucket.Insert(id, &beer, 0)
		http.Redirect(w, r, "/beers/show/"+id, http.StatusFound)
	}
}

func beerEditHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.Split(r.URL.Path, "/")[3]

	if r.Method != "POST" {
		var beer BeerFull
		if _, err := bucket.Get(id, &beer); err != nil {
			fmt.Fprintf(w, "Get Error: %v\n", err)
			return
		}

		data := tdBeerEdit{
			IsCreate: false,
			Beer:     beer,
		}
		executeTemplate(w, "beer/edit.html", data)
	} else {
		r.ParseForm()
		beer := beerFromForm(r.Form)
		beer.Type = "beer"

		bucket.Upsert(id, &beer, 0)
		http.Redirect(w, r, "/beers/show/"+id, http.StatusFound)
	}
}

type breweryByNameRow struct {
	Id  string
	Key string
}

type tdBrewIndex struct {
	Results []Brewery
}

func brewIndexHandler(w http.ResponseWriter, r *http.Request) {
	vq := gocb.NewViewQuery("brewery", "by_name").Limit(entriesPerPage)
	rows, err := bucket.ExecuteViewQuery(vq)
	if nil != err {
		panic(err)
	}
	var row breweryByNameRow
	var breweries []Brewery
	for rows.Next(&row) {
		breweries = append(breweries, Brewery{
			Id:   row.Id,
			Name: row.Key,
		})
	}

	data := tdBrewIndex{
		Results: breweries,
	}
	executeTemplate(w, "brewery/index.html", data)
}

func brewSearchHandler(w http.ResponseWriter, r *http.Request) {
	value := r.URL.Query().Get("value")

	vq := gocb.NewViewQuery("brewery", "by_name").Limit(entriesPerPage)
	vq.Range(value, value+"\u0FFFF", false)

	rows, err := bucket.ExecuteViewQuery(vq)
	if nil != err {
		panic(err)
	}

	var row breweryByNameRow
	var breweries []Brewery
	for rows.Next(&row) {
		breweries = append(breweries, Brewery{
			Id:   row.Id,
			Name: row.Key,
		})
	}

	bytes, err := json.Marshal(breweries)
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}

	w.Write(bytes)
}

type tdBrewShow struct {
	Brewery       BreweryFull
	BreweryFields map[string]string
}

func brewShowHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.Split(r.URL.Path, "/")[3]

	var brew BreweryFull
	if _, err := bucket.Get(id, &brew); err != nil {
		fmt.Fprintf(w, "Get Error: %v\n", err)
		return
	}

	brew.Id = id
	data := tdBrewShow{
		Brewery: brew,
		BreweryFields: map[string]string{
			"name":        brew.Name,
			"description": brew.Description,
			"city":        brew.City,
			"state":       brew.State,
			"code":        brew.Code,
			"country":     brew.Country,
			"phone":       brew.Phone,
			"website":     brew.Website,
		},
	}
	executeTemplate(w, "brewery/show.html", data)
}

func main() {
	cluster, _ := gocb.Connect("couchbase://127.0.0.1")
	bucket, _ = cluster.OpenBucket("beer-sample", "")

	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/beers", beerIndexHandler)
	http.HandleFunc("/beers/search", beerSearchHandler)
	http.HandleFunc("/beers/show/", beerShowHandler)
	http.HandleFunc("/beers/create", beerCreateHandler)
	http.HandleFunc("/beers/edit/", beerEditHandler)
	http.HandleFunc("/beers/delete/", removeHandler)
	http.HandleFunc("/breweries", brewIndexHandler)
	http.HandleFunc("/breweries/search", brewSearchHandler)
	http.HandleFunc("/breweries/show/", brewShowHandler)
	http.HandleFunc("/breweries/delete/", removeHandler)

	http.Handle("/css/", http.FileServer(http.Dir("static/")))
	http.Handle("/js/", http.FileServer(http.Dir("static/")))

	fmt.Printf("Starting server on :9980\n")
	http.ListenAndServe(":9980", nil)
}
