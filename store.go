package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
)

type URLStore struct {
	urls map[string]string //map from short to long URLS
	mu   sync.RWMutex
	save chan record
}

type record struct {
	key, URL string
}

const saveQueueLength = 1000

func (s *URLStore) load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening URLStore:", err)
		return err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	for err == nil {
		var r record
		if err = d.Decode(&r); err == nil {
			s.Set(r.key, r.URL)
		}
	}
	if err == io.EOF {
		return nil
	}
	// error occurred:
	log.Println("Error decoding URLStore:", err) // map hasn't been read correctly
	return err
}

func (s *URLStore) saveLoop(filename string) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("URLStore:", err)
	}
	defer f.Close()
	e := json.NewEncoder(f)
	for {
		r := <-s.save
		if err := e.Encode(r); err != nil {
			log.Println("Error saving to URLStore: ", err)
		}
	}
}

func (s *URLStore) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.urls[key]
}

func (s *URLStore) Set(key, url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, present := s.urls[key]
	if present {
		return false
	}
	s.urls[key] = url
	return true
}

func (s *URLStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.urls)
}

func (s *URLStore) Put(url string) string {
	for {
		key := genKey(s.Count())
		if s.Set(key, url) {
			s.save <- record{key, url}
			return key
		}
	}
	panic("shouldn't get here")
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{
		urls: make(map[string]string),
		save: make(chan record, saveQueueLength),
	}
	if err := s.load(filename); err != nil {
		log.Println("error loading URLStore:", err)
	}
	go s.saveLoop(filename)
	return s
}
