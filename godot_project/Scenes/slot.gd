extends Button

@onready var audio_stream_player = $AudioStreamPlayer

# Called when the node enters the scene tree for the first time.
func _ready():
	#var bp = BuildingPlacer
	pass # Replace with function body.

# Called every frame. 'delta' is the elapsed time since the previous frame.
func _process(delta):
	pass

func _on_button_up():
	print("button up")
	audio_stream_player.play()
	#/BuildingPlacer.Instance.toggle_build_mode()
