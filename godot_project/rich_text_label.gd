extends RichTextLabel

var count = 0
var json_path = "res://game_data.json"  # Path relative to the project directory
var go_script_path = "res://go_script/main.go"  # Path to the Go script

func _ready():
	print("RichTextLabel script starting")
	
	# Load data from JSON file
	load_from_json_file()
	
	# Update display
	update_display()
	
	# Set up a timer to periodically check for file changes
	var timer = Timer.new()
	timer.wait_time = 1.0  # Check every second
	timer.autostart = true
	timer.timeout.connect(_on_timer_timeout)
	add_child(timer)

func _on_timer_timeout():
	# Periodically check if the JSON file has been updated externally
	load_from_json_file()
	update_display()

func load_from_json_file():
	print("Attempting to load from JSON file: ", json_path)
	
	# Check if the file exists
	if FileAccess.file_exists(json_path):
		print("JSON file exists, reading data")
		
		var file = FileAccess.open(json_path, FileAccess.READ)
		if file:
			var json_string = file.get_as_text()
			file.close()
			
			# Parse JSON
			var json = JSON.new()
			var error = json.parse(json_string)
			
			if error == OK:
				var data = json.data
				if data.has("money_count"):
					count = data["money_count"]
					print("Successfully loaded money_count: ", count)
				else:
					print("JSON doesn't contain money_count key")
			else:
				print("JSON parse error: ", json.get_error_message(), " at line ", json.get_error_line())
		else:
			print("Couldn't open file for reading")
	else:
		print("JSON file doesn't exist at path: ", json_path)
		
		# Create default file if it doesn't exist
		var default_data = {"money_count": 0}
		save_to_json_file(default_data)

func save_to_json_file(data):
	print("Saving data to JSON file: ", json_path)
	
	var file = FileAccess.open(json_path, FileAccess.WRITE)
	if file:
		var json_string = JSON.stringify(data, "  ")
		file.store_string(json_string)
		file.close()
		print("Successfully saved data to JSON file")
	else:
		print("Couldn't open file for writing")

func update_display():
	# Update the text display
	text = "Money: " + str(count) 
	print("Display updated: Money = ", count)

func _on_button_button_down():
	# Button press handler
	count += 1
	update_display()
	
	# Save updated value to JSON
	save_to_json_file({"money_count": count, "last_updated": Time.get_datetime_string_from_system()})

func fetch_from_mongodb():
	print("Fetching data from MongoDB via Go script...")
	
	# Get the project directory path
	var project_dir = OS.get_executable_path().get_base_dir()
	var go_script_dir = project_dir.path_join("go_script")
	
	# Execute the Go script with "fetch" command
	var output = []
	var exit_code = OS.execute("go", ["run", "main.go", "fetch"], output, true, go_script_dir)
	
	if exit_code == 0:
		print("MongoDB fetch successful")
		print("Output: ", output)
		# Force reload from JSON after Go script runs
		load_from_json_file()
		update_display()
	else:
		print("Error fetching from MongoDB. Exit code: ", exit_code)
		print("Output: ", output)

func save_to_mongodb():
	print("Saving data to MongoDB via Go script...")
	
	# Get the project directory path
	var project_dir = OS.get_executable_path().get_base_dir()
	var go_script_dir = project_dir.path_join("go_script")
	
	# Execute the Go script with "save" command and current count
	var output = []
	var exit_code = OS.execute("go", ["run", "main.go", "save", str(count)], output, true, go_script_dir)
	
	if exit_code == 0:
		print("MongoDB save successful")
		print("Output: ", output)
	else:
		print("Error saving to MongoDB. Exit code: ", exit_code)
		print("Output: ", output)

func _on_run_go_script_button_pressed():
	# Connect this to a button for running the Go script
	fetch_from_mongodb()

func _on_update_button_button_down():
	# First update the local count
	count += 1
	update_display()
	
	# Then save to both JSON and MongoDB
	save_to_json_file({"money_count": count, "last_updated": Time.get_datetime_string_from_system()})
	save_to_mongodb()
