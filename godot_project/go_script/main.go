package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GameData represents the structure of our JSON file and MongoDB document
type GameData struct {
	MoneyCount  int    `json:"money_count" bson:"money_count"`
	LastUpdated string `json:"last_updated,omitempty" bson:"last_updated,omitempty"`
}

func main() {
	// Check command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [fetch|save] [count]")
		os.Exit(1)
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// If we're in the go_script directory, go up one level
	if filepath.Base(currentDir) == "go_script" {
		currentDir = filepath.Dir(currentDir)
	}

	// Path to the JSON file
	jsonFilePath := filepath.Join(currentDir, "game_data.json")

	// Command from argument
	command := os.Args[1]

	// Connect to MongoDB
	client, err := connectToMongoDB()
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer client.Disconnect(context.Background())

	// Get collection
	collection := client.Database("gamedb").Collection("player_data")

	// Process command
	switch command {
	case "fetch":
		// Fetch data from MongoDB and update the JSON file
		err = fetchFromMongoDB(collection, jsonFilePath)
	case "save":
		if len(os.Args) < 3 {
			fmt.Println("Save command requires a count parameter")
			os.Exit(1)
		}
		count, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Invalid count value: %v\n", err)
			os.Exit(1)
		}
		err = saveToMongoDB(collection, jsonFilePath, count)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func connectToMongoDB() (*mongo.Client, error) {
	// Set client options - update with your MongoDB connection string
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to MongoDB!")
	return client, nil
}

func fetchFromMongoDB(collection *mongo.Collection, jsonFilePath string) error {
	// Set context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query MongoDB for player data
	// Assuming we're looking for a specific player - you may need to customize this
	filter := bson.M{"player_id": "default"} // Modify with your actual player ID field

	// Find the document
	var result GameData
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No document found, create default data
			result = GameData{
				MoneyCount:  0,
				LastUpdated: time.Now().Format(time.RFC3339),
			}

			// Insert default data into MongoDB
			_, err = collection.InsertOne(ctx, bson.M{
				"player_id":    "default",
				"money_count":  result.MoneyCount,
				"last_updated": result.LastUpdated,
			})
			if err != nil {
				return fmt.Errorf("failed to insert default data: %v", err)
			}
		} else {
			return fmt.Errorf("error finding document: %v", err)
		}
	}

	fmt.Printf("Fetched data from MongoDB: money_count = %d\n", result.MoneyCount)

	// Write the data to the JSON file
	return writeGameData(jsonFilePath, result)
}

func saveToMongoDB(collection *mongo.Collection, jsonFilePath string, count int) error {
	// Set context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create game data object
	gameData := GameData{
		MoneyCount:  count,
		LastUpdated: time.Now().Format(time.RFC3339),
	}

	// Update MongoDB
	filter := bson.M{"player_id": "default"} // Modify with your actual player ID field
	update := bson.M{
		"$set": bson.M{
			"money_count":  gameData.MoneyCount,
			"last_updated": gameData.LastUpdated,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update MongoDB: %v", err)
	}

	fmt.Printf("Saved to MongoDB: money_count = %d\n", count)

	// Write the same data to the JSON file
	return writeGameData(jsonFilePath, gameData)
}

func writeGameData(filePath string, data GameData) error {
	// Convert to JSON with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error converting to JSON: %v", err)
	}

	// Write to file
	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	fmt.Println("Successfully updated JSON file")
	return nil
}
