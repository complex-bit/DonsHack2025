extends CharacterBody3D

@onready var navigation_agent_3d: NavigationAgent3D = $NavigationAgent3D
# Reference to a node that will be rotated for looking (assumes there's a child node for this)
@onready var look_pivot = $LookPivot # Make sure to add this node in your scene

# Movement parameters - reduced speed for slower movement
var max_speed = 0.8
var acceleration = 1.2
var deceleration = 1.5
var arrival_threshold = 0.5

# Wobble parameters
var wobble_time = 0.0
var wobble_speed = 2.0
var wobble_intensity_pos = 0.06  # Reduced for subtler movement
var wobble_intensity_rot = 0.15  # Reduced for subtler rotation
var height_offset = 0.0
var rotation_speed = 4.0  # Controls how quickly the character rotates to face direction

# Timer variables for random movement
var time_until_next_walk: float = 0.0
var min_walk_interval: float = 2.0 # Increased for more idle time
var max_walk_interval: float = 8.0 # Increased for more idle time
var current_velocity = Vector3.ZERO

# Idle look-around variables
var look_around_time = 0.0
var look_speed = 0.7
var look_intensity = 0.8
var current_look_target = Vector3.ZERO
var look_change_timer = 0.0

func _ready() -> void:
	# Set initial timer for first random walk
	randomize()
	time_until_next_walk = randf_range(min_walk_interval, max_walk_interval)
	look_change_timer = randf_range(1.0, 3.0)
	
	# Configure the navigation agent
	navigation_agent_3d.path_desired_distance = 0.5
	navigation_agent_3d.target_desired_distance = 0.5

func _unhandled_input(event: InputEvent) -> void:
	if event.is_action_pressed("ui_accept"):
		set_random_target()

func set_random_target() -> void:
	var random_position := Vector3.ZERO
	random_position.x = randf_range(-5.0, 5.0)
	random_position.z = randf_range(-5.0, 5.0)
	navigation_agent_3d.set_target_position(random_position)

func update_look_direction(delta: float, moving: bool) -> void:
	# Update look around timer
	look_around_time += delta * look_speed
	look_change_timer -= delta
	
	if look_change_timer <= 0 or moving:
		# Set new random look target if idle, or look at movement direction if moving
		if moving:
			# Look in the direction of movement
			if current_velocity.length() > 0.1:
				current_look_target = current_velocity.normalized()
			look_change_timer = 0.2 # Quick updates while moving
		else:
			# Random looking around when idle
			current_look_target = Vector3(
				randf_range(-1.0, 1.0),
				randf_range(-0.2, 0.5), # Mostly look forward/slightly up
				randf_range(-1.0, 1.0)
			).normalized()
			look_change_timer = randf_range(1.0, 3.0)
	
	# Only apply rotation if we have a look pivot
	if is_instance_valid(look_pivot):
		var target_rotation = Quaternion.IDENTITY
		
		if moving:
			# Smoothly look in movement direction
			var look_at_pos = global_position + current_look_target
			look_pivot.look_at(look_at_pos, Vector3.UP)
		else:
			# Add some wobble to idle looking
			var wobble_factor = sin(look_around_time * 1.5) * 0.3
			var target_pos = global_position + current_look_target + Vector3(wobble_factor, 0, wobble_factor)
			look_pivot.look_at(target_pos, Vector3.UP)
			
			# Add tilting like it's curious
			look_pivot.rotation.z = sin(look_around_time * 0.8) * 0.2

func _physics_process(delta: float) -> void:
	# Update wobble time
	wobble_time += delta * wobble_speed
	
	# Update timer for random walking
	time_until_next_walk -= delta
	
	# If timer has expired, set a new random target and reset timer
	if time_until_next_walk <= 0:
		set_random_target()
		time_until_next_walk = randf_range(min_walk_interval, max_walk_interval)
	
	var is_moving = false
	
	if navigation_agent_3d.is_navigation_finished():
		# Smoothly decelerate to a stop
		current_velocity = current_velocity.move_toward(Vector3.ZERO, deceleration * delta)
	else:
		is_moving = true
		var destination = navigation_agent_3d.get_next_path_position()
		var direction = (destination - global_position).normalized()
		
		# Calculate distance to the target
		var distance_to_target = global_position.distance_to(destination)
		
		if distance_to_target <= arrival_threshold:
			# Start decelerating as we approach the target
			current_velocity = current_velocity.move_toward(Vector3.ZERO, deceleration * delta)
			is_moving = false
		else:
			# Calculate target velocity based on direction and speed
			var target_velocity = direction * max_speed
			
			# Add some wobble to the movement direction
			target_velocity.x += sin(wobble_time * 1.2) * wobble_intensity_pos
			target_velocity.z += cos(wobble_time * 1.5) * wobble_intensity_pos
			
			# Smoothly accelerate towards the target velocity
			current_velocity = current_velocity.move_toward(target_velocity, acceleration * delta)
			
			# If we're getting close to the target, start slowing down
			if distance_to_target < 1.0:
				var slowdown_factor = clamp(distance_to_target / 1.0, 0.3, 1.0)
				current_velocity *= slowdown_factor
	
	# Apply wobble to vertical position for a bouncy effect - more subtle when moving
	var speed_factor = current_velocity.length() / max_speed
	height_offset = sin(wobble_time * 2.5) * wobble_intensity_pos * (0.8 + speed_factor * 0.2)
	
	# Apply the calculated velocity with wobble
	velocity = current_velocity
	velocity.y = height_offset * 2.0 # Reduced multiplier for subtler vertical movement
	
	# Update the look direction
	update_look_direction(delta, is_moving)
	
	# Handle character rotation to face movement direction
	if is_moving and current_velocity.length() > 0.05:
		# Get the target rotation to face movement direction
		var target_rotation = Vector3.ZERO
		var look_direction = current_velocity.normalized()
		
		# Only rotate around Y axis (up/down)
		target_rotation.y = atan2(look_direction.x, look_direction.z)
		
		# Smoothly interpolate current rotation to target rotation
		rotation.y = lerp_angle(rotation.y, target_rotation.y, delta * rotation_speed)
		
		# Keep a slight tilt for character personality, but reduced while moving
		rotation.x = sin(wobble_time * 0.2) * 0.02
	else:
		# Add slight wobble to the base rotation when idle
		rotation.y = sin(wobble_time * 0.3) * 0.1 + rotation.y
		rotation.x = sin(wobble_time * 0.2) * 0.05
	
	move_and_slide()
