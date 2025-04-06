extends CharacterBody3D

@onready var navigation_agent_3d: NavigationAgent3D = $NavigationAgent3D
var speed = 2.5
var max_speed = 1.5
var acceleration = 2.0
var deceleration = 3.0
var arrival_threshold = 0.5  # Distance at which we consider "arrived"
#sin wave for z axis 
# Timer variables for random movement
var time_until_next_walk: float = 0.0
var min_walk_interval: float = 1.0
var max_walk_interval: float = 7.0
var current_velocity = Vector3.ZERO

func _ready() -> void:
	# Set initial timer for first random walk
	randomize()
	time_until_next_walk = randf_range(min_walk_interval, max_walk_interval)
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

func _physics_process(delta: float) -> void:
	# Update timer for random walking
	time_until_next_walk -= delta
	
	# If timer has expired, set a new random target and reset timer
	if time_until_next_walk <= 0:
		set_random_target()
		time_until_next_walk = randf_range(min_walk_interval, max_walk_interval)
	
	if navigation_agent_3d.is_navigation_finished():
		# Smoothly decelerate to a stop
		current_velocity = current_velocity.move_toward(Vector3.ZERO, deceleration * delta)
	else:
		var destination = navigation_agent_3d.get_next_path_position()
		var direction = (destination - global_position).normalized()
		
		# Calculate distance to the target
		var distance_to_target = global_position.distance_to(destination)
		
		if distance_to_target <= arrival_threshold:
			# Start decelerating as we approach the target
			current_velocity = current_velocity.move_toward(Vector3.ZERO, deceleration * delta)
		else:
			# Calculate target velocity based on direction and speed
			var target_velocity = direction * max_speed
			
			# Smoothly accelerate towards the target velocity
			current_velocity = current_velocity.move_toward(target_velocity, acceleration * delta)
			
			# If we're getting close to the target, start slowing down
			if distance_to_target < 1.0:
				var slowdown_factor = clamp(distance_to_target / 1.0, 0.3, 1.0)
				current_velocity *= slowdown_factor
	
	# Apply the calculated velocity
	velocity = current_velocity
	move_and_slide()
