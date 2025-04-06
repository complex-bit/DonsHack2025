extends Node3D

@export var rotation_speed: float = 2.0
# Flip these values since the camera seems to be inverted
@export var min_vertical_angle: float = -70.0  # Looking up limit (negative in Godot means looking up)
@export var max_vertical_angle: float = -10.0    # Looking down limit (positive means looking down)
@onready var spring_arm: SpringArm3D = $SubViewportContainer/SubViewport/SpringArm3D

# Current rotation values
var rotation_x: float = deg_to_rad(-25.0)  # Start at -25 degrees (looking down at island)
var rotation_y: float = 0.0

# Target rotation values for smoothing
var target_rotation_x: float = deg_to_rad(-25.0)
var target_rotation_y: float = 0.0

# Zoom parameters
@export var zoom_speed: float = 0.5
@export var min_spring_length: float = 1.0
@export var max_spring_length: float = 8.0
@export var initial_spring_length: float = 3.0

# Smoothing parameters
@export var rotation_smoothing: float = 10.0  # Higher value = smoother but slower rotation
@export var zoom_smoothing: float = 5.0       # Higher value = smoother but slower zoom
var target_spring_length: float = initial_spring_length

func _ready():
	# Initialize with a predefined camera angle
	rotation_x = deg_to_rad(-25.0)
	target_rotation_x = rotation_x
	spring_arm.rotation.x = rotation_x
	
	rotation_y = 0.0
	target_rotation_y = rotation_y
	spring_arm.rotation.y = rotation_y
	
	# Set initial spring arm length
	spring_arm.spring_length = initial_spring_length
	target_spring_length = initial_spring_length
	
	print("Spring arm rotation script initialized")
	print("Initial rotation X: ", rad_to_deg(rotation_x), " Y: ", rad_to_deg(rotation_y))
	print("Initial spring length: ", spring_arm.spring_length)

func _physics_process(delta: float):
	# Process keyboard input for rotation
	if Input.is_action_pressed("ui_left"):
		target_rotation_y -= rotation_speed * delta
	if Input.is_action_pressed("ui_right"):
		target_rotation_y += rotation_speed * delta
	if Input.is_action_pressed("ui_up"):
		target_rotation_x -= rotation_speed * delta
	if Input.is_action_pressed("ui_down"):
		target_rotation_x += rotation_speed * delta
	
	# Clamp target vertical rotation
	target_rotation_x = clamp(target_rotation_x, deg_to_rad(min_vertical_angle), deg_to_rad(max_vertical_angle))
	
	# Apply smoothing to rotations
	rotation_x = lerp(rotation_x, target_rotation_x, delta * rotation_smoothing)
	rotation_y = lerp(rotation_y, target_rotation_y, delta * rotation_smoothing)
	
	# Apply smoothing to zoom
	spring_arm.spring_length = lerp(spring_arm.spring_length, target_spring_length, delta * zoom_smoothing)
	
	# Apply rotations to spring arm
	spring_arm.rotation.x = rotation_x
	spring_arm.rotation.y = rotation_y
	
	# Optional debug info
	# print("Rotation X: ", rad_to_deg(rotation_x), " Y: ", rad_to_deg(rotation_y))

## Input handling for mouse controls and zoom
#func _input(event):
	#if event is InputEventMouseButton:
		#if event.button_index == MOUSE_BUTTON_WHEEL_UP:
			## Zoom in (decrease spring length)
			#target_spring_length = max(min_spring_length, target_spring_length - zoom_speed)
		#elif event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
			## Zoom out (increase spring length)
			#target_spring_length = min(max_spring_length, target_spring_length + zoom_speed)
