package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func read_json() ([]map[string]interface{}, error) {
	// Open the JSON file
	file, err := os.Open("exits.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file contents into a byte slice
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Declare a slice of maps to hold the unmarshalled JSON
	var data []map[string]interface{}

	// Unmarshal the byte slice into the slice of maps
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// CORS middleware to handle preflight requests
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Proceed to the next handler
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Define path to static directory (relative from Backend/)
	staticDir := filepath.Join("..", "static")

	// Endpoint for test data
	var testValue = 100 // Store the value in a variable that can be modified

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "POST" {
			// Handle POST request to increment value
			testValue += 5
		}

		data := map[string]interface{}{
			"test": testValue,
		}
		json.NewEncoder(w).Encode(data)
	})

	// New endpoint for assignments
	http.HandleFunc("/assignments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		data, err := read_json()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(data)
	})

	// Serve static files at /static/
	fs := http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir)))
	http.Handle("/static/", enableCORS(fs))

	// Serve main.html at /
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticDir, "main.html"))
	})

	// Print assignments at startup for debugging
	data, err := read_json()
	if err != nil {
		log.Fatalf("Error reading JSON: %v", err)
	}

	fmt.Println("Loaded assignments:")
	for _, entry := range data {
		fmt.Println(entry)
	}

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
