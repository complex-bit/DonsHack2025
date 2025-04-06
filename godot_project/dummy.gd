extends Node3D

# Store the node reference statically so it's accessible from the callback
static var _instance: Node3D
@onready var rich_text_label = $Control/PanelContainer/RichTextLabel

func _ready():
	_instance = self
	
	var fileLoadCallback = JavaScriptBridge.create_callback(FileParser)
	
	if OS.has_feature("web"):
		var window = JavaScriptBridge.get_interface("window")
		window.getFile(fileLoadCallback)

func _on_upload_button_down() -> void:
	if OS.has_feature("web"):
		var window = JavaScriptBridge.get_interface("window")
		window.getFile(JavaScriptBridge.create_callback(FileParser))

static func FileParser(args):
	if _instance and _instance.rich_text_label:
		# Parse the JSON
		var json_data = JSON.parse_string(args[0])
		
		# Extract just the money amount
		if json_data and json_data.has("test"):
			var money_amount = json_data["test"]
			
			# Format as currency
			var formatted_money = "$" + str(money_amount)
			
			# Update the RichTextLabel with just the money
			_instance.rich_text_label.text = formatted_money
		else:
			# Fallback if the data doesn't have the expected format
			_instance.rich_text_label.text = "No money data found"
