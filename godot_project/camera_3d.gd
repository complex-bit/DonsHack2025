extends SpringArm3D

# Rotation speed (degrees per second)
@export var rotation_speed: float = 100.0

# Vertical rotation limits (in degrees)
@export var min_elevation: float = -15.0    # Prevent looking below ground
@export var max_elevation: float = 80.0     # Max upward angle

# Distance settings
@export var min_spring_length: float = 5.0
@export var max_spring_length: float = 20.0
@export var default_spring_length: float = 15.0

# Mouse sensitivity
@export var mouse_sensitivity: float = 0.5

# Whether to use keyboard for rotation
@export var use_keyboard: bool = true

# Current rotation values in degrees
var horizontal_rotation: float = 0.0
var vertical_rotation: float = 0.0

# Helper function to find camera child
func _find_camera_child() -> Camera3D:
	for child in get_children():
		if child is Camera3D:
			return child
	return null

func _ready() -> void:
	# Initialize spring length
	spring_length = default_spring_length
	
	# Keep local position at zero relative to parent
	position = Vector3.ZERO
	
	# Find the camera child if it exists
	var camera = _find_camera_child()
	if camera:
		# Point the camera toward the center with 10 degree offset
		camera.rotation_degrees = Vector3(10, 0, 0)
	
	# Set initial arm rotation
	_update_arm_rotation()
	
func _process(delta: float) -> void:
	# Keep local position at zero relative to parent
	position = Vector3.ZERO
	
	# Make sure camera stays pointed toward origin with 10 degree offset
	var camera = _find_camera_child()
	if camera:
		camera.rotation_degrees = Vector3(10, 0, 0)
		
	var input_dir = Vector2.ZERO
	
	# Handle keyboard input
	if use_keyboard:
		if Input.is_action_pressed("ui_left"):
			input_dir.x -= 1
		if Input.is_action_pressed("ui_right"):
			input_dir.x += 1
		if Input.is_action_pressed("ui_up"):
			input_dir.y += 1
		if Input.is_action_pressed("ui_down"):
			input_dir.y -= 1
	
	# Apply rotation based on keyboard input
	if input_dir != Vector2.ZERO:
		horizontal_rotation -= input_dir.x * rotation_speed * delta
		vertical_rotation += input_dir.y * rotation_speed * delta
		
		# Apply constraints and update
		_constrain_rotation()
		_update_arm_rotation()

func _unhandled_input(event: InputEvent) -> void:
	# Handle mouse movement for arm rotation
	if event is InputEventMouseMotion and Input.is_mouse_button_pressed(MOUSE_BUTTON_LEFT):
		horizontal_rotation -= event.relative.x * mouse_sensitivity * 0.1
		vertical_rotation -= event.relative.y * mouse_sensitivity * 0.1
		
		# Apply constraints and update
		_constrain_rotation()
		_update_arm_rotation()
	
	# Handle mouse wheel for zoom
	if event is InputEventMouseButton:
		if event.button_index == MOUSE_BUTTON_WHEEL_UP:
			spring_length = max(spring_length - 0.5, min_spring_length)
		elif event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
			spring_length = min(spring_length + 0.5, max_spring_length)

func _constrain_rotation() -> void:
	# Clamp vertical rotation to prevent looking below ground
	vertical_rotation = clamp(vertical_rotation, min_elevation, max_elevation)
	
	# Keep horizontal rotation within 0-360 degree range
	while horizontal_rotation > 360:
		horizontal_rotation -= 360
	while horizontal_rotation < 0:
		horizontal_rotation += 360

func _update_arm_rotation() -> void:
	# Set the rotation directly using Euler angles
	rotation_degrees = Vector3(
		-vertical_rotation,  # X rotation (pitch)
		horizontal_rotation, # Y rotation (yaw)
		0                    # Z rotation (roll)
	)
