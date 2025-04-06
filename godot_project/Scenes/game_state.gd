extends Node3D

# Path for saving/loading the game data
const SAVE_PATH = "user://building_save.tres"

func _ready():
	print("Game state initialized")

func save_game(building_data: Array) -> void:
	# Create a new save resource
	var save_data = BuildingSaveData.new()
	save_data.buildings = building_data
	
	# Save the resource to disk
	var result = ResourceSaver.save(save_data, SAVE_PATH)
	
	if result == OK:
		print("Game saved successfully to: " + SAVE_PATH)
	else:
		print("Failed to save game: Error code " + str(result))
	
func load_game() -> Array:
	# Check if save file exists
	if not FileAccess.file_exists(SAVE_PATH):
		print("No save file found at: " + SAVE_PATH)
		return []
	
	# Load the resource
	var save_data = ResourceLoader.load(SAVE_PATH)
	
	if save_data is BuildingSaveData:
		print("Game loaded successfully from: " + SAVE_PATH)
		return save_data.buildings
	else:
		print("Failed to load game or invalid save data")
		return []
