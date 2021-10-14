package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"net/url"
	"program/joker"
	"program/model"

	"program/storage"
	"program/storage/filestorage"
)

func main() {

	storage.St = &filestorage.FileStorage{}

	//file:= filestorage.NewFileStorage("reddit_jokes.json")
	//fmt.Println(file)

	myRouter := handleRequest(&apiHandler{})

	err := http.ListenAndServe(":9090", myRouter)
	if err != nil {
		panic(err)
	}

}

type apiHandler struct {
	server joker.Server
}

func handleRequest(h *apiHandler) *mux.Router {
	myRouter := mux.NewRouter().StrictSlash(true)
	//myRouter.HandleFunc("/jokes", homePage).Methods("GET")
	myRouter.HandleFunc("/jokes/method/save", h.Save)
	myRouter.HandleFunc("/jokes/method/load", h.Load)
	myRouter.HandleFunc("/jokes/funniest", h.getFunniestJokes)
	myRouter.HandleFunc("/jokes/random", h.getRandomJoke)
	myRouter.HandleFunc("/jokes", h.addJoke).Methods("POST")
	myRouter.HandleFunc("/jokes/{id}", h.getJokeByID)
	myRouter.HandleFunc("/jokes/search/{text}", h.getJokeByText)

	return myRouter
}

func (h *apiHandler) Load(w http.ResponseWriter, r *http.Request) {

	h.server.JStruct()
	json.NewEncoder(w).Encode("File loaded")
}

func (h *apiHandler) getJokeByID(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]
	res, err := h.server.ID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *apiHandler) getJokeByText(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	text := vars["text"]
	res, err := h.server.Text(text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&res)

}

func (h *apiHandler) getFunniestJokes(w http.ResponseWriter, r *http.Request) {

	m, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Fatal(err)
	}
	res, err := h.server.Funniest(m)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(res)

}

func (h *apiHandler) getRandomJoke(w http.ResponseWriter, r *http.Request) {

	m, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Fatal(err, "Error parsing query")
	}

	res, err := h.server.Random(m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(res)

}

func (h *apiHandler) addJoke(w http.ResponseWriter, r *http.Request) {
	type serverError struct {
		Code        string
		Description string
	}

	var j model.Joke
	err := json.NewDecoder(io.LimitReader(r.Body, 4*1024)).Decode(&j)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = model.Joke.Validate(j)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(serverError{
			Code:        "validation_err",
			Description: err.Error(),
		})
		return
	}
	res, err1 := h.server.Add(j)
	if err1 != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&res)
}

func (h *apiHandler) Save(w http.ResponseWriter, r *http.Request) {

	err := storage.St.Save(h.server.JStruct())
	if err != nil {
		http.Error(w, "error saving file", 500)
	}
	json.NewEncoder(w).Encode("File saved")

}
