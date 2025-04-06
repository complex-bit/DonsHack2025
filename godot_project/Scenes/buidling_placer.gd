extends Node3D

@onready var area: Area3D = $Area3D
@onready var area_colshape: CollisionShape3D = $Area3D/CollisionShape3D
@onready var model: MeshInstance3D = $building_preview
@onready var ground_placements = $"../ground_placements"
@onready var game_state = $"../game_state"

var max_distance_threshold = 0.5  # Maximum distance from ground to consider valid
var ground_offset = 0.1  # Height offset for preview
var placement_offset = 0.5  # Higher offset for final placement to avoid sinking
var rotation_speed = 15.0  # Degrees per scroll event

# List of available building types
var all_buildings = ["res://house_6.tscn","res://buildings/cherry_blossom.tscn"]
var current_building_index = 0

# Track all placed buildings
var placed_buildings = []

# Build mode toggle
var build_mode = true

func model_red() -> void:
	if model:
		model.set("instance_shader_parameters/instance_color_01", Color("red"))

func model_blue() -> void:
	if model:
		model.set("instance_shader_parameters/instance_color_01", Color("blue"))
	
func enable_area() -> void:
	if area_colshape:
		area_colshape.disabled = false
	
func disable_area() -> void:
	if area_colshape:
		area_colshape.disabled = true

func change_building_type(index: int) -> void:
	if index >= 0 and index < all_buildings.size():
		current_building_index = index
		update_preview_model()
		print("Switched to building type: " + all_buildings[current_building_index])

func is_overlapping_with_placed_buildings() -> bool:
	# Create an AABB for the current preview building
	var preview_aabb = get_aabb()
	
	# Check for overlaps with all placed buildings
	for building in placed_buildings:
		if building == null or not is_instance_valid(building):
			continue
			
		var building_aabb = get_node_aabb(building)
		
		# Check if AABBs intersect
		if preview_aabb.intersects(building_aabb):
			#print("Overlapping with existing building")
			return true
	
	return false

func get_aabb() -> AABB:
	# Create an AABB based on the model's bounds
	var aabb = AABB()
	aabb.position = global_position
	
	# Use the shape's size for the AABB
	var shape = area_colshape.shape
	var extents = Vector3(1, 1, 1)
	
	if shape is BoxShape3D:
		extents = shape.size * 0.35  # 70% of size
	elif shape is ConvexPolygonShape3D:
		var shape_aabb = AABB()
		for point in shape.points:
			shape_aabb = shape_aabb.expand(point * 0.7)
		extents = shape_aabb.size * 0.5
	
	# Apply rotation to the extents
	var basis = global_transform.basis
	extents = Vector3(
		abs(basis.x.x * extents.x) + abs(basis.x.y * extents.y) + abs(basis.x.z * extents.z),
		abs(basis.y.x * extents.x) + abs(basis.y.y * extents.y) + abs(basis.y.z * extents.z),
		abs(basis.z.x * extents.x) + abs(basis.z.y * extents.y) + abs(basis.z.z * extents.z)
	)
	
	aabb.size = extents * 2
	return aabb

func get_node_aabb(node: Node3D) -> AABB:
	# Create an AABB for a placed building
	var aabb = AABB()
	aabb.position = node.global_position
	
	# Estimate size based on node scale
	var extents = node.scale * 0.5
	
	# Apply rotation to the extents
	var basis = node.global_transform.basis
	extents = Vector3(
		abs(basis.x.x * extents.x) + abs(basis.x.y * extents.y) + abs(basis.x.z * extents.z),
		abs(basis.y.x * extents.x) + abs(basis.y.y * extents.y) + abs(basis.y.z * extents.z),
		abs(basis.z.x * extents.x) + abs(basis.z.y * extents.y) + abs(basis.z.z * extents.z)
	)
	
	aabb.size = extents * 2
	return aabb

func placement_check() -> bool:
	model_red() # Default to red (invalid placement)
	
	# Check for overlap with existing buildings
	if is_overlapping_with_placed_buildings():
		return false
		
	# Get ground plane position
	var ground_y = ground_placements.global_position.y
	var ground_top = ground_y + (ground_placements.scale.y / 2) if ground_placements is CSGBox3D else ground_y
	
	# Get the bounds of our shape
	var shape = area_colshape.shape
	var bounds = Vector3(0.7, 0.7, 0.7) # Smaller size - 70% of original
	
	# Handle different collision shape types
	if shape is BoxShape3D:
		bounds = shape.size * 0.35  # 70% of half-size
	elif shape is ConvexPolygonShape3D:
		var aabb = AABB()
		for point in shape.points:
			aabb = aabb.expand(point * 0.7)  # Scale points by 70%
		bounds = aabb.size * 0.35
	
	# Define the four corner points
	var points_to_check: Array = [
		area_colshape.global_position + Vector3(bounds.x, -bounds.y, bounds.z),
		area_colshape.global_position + Vector3(bounds.x, -bounds.y, -bounds.z),
		area_colshape.global_position + Vector3(-bounds.x, -bounds.y, -bounds.z),
		area_colshape.global_position + Vector3(-bounds.x, -bounds.y, bounds.z)
	]
	
	var valid_ground_points = 0
	
	# Check each corner point's distance to ground top surface
	for point in points_to_check:
		var distance_to_ground = abs(point.y - ground_top)
		
		if distance_to_ground <= max_distance_threshold:
			valid_ground_points += 1
	
	# If all four points are close enough to the ground, change color to blue
	if valid_ground_points == 4:
		model_blue()
		return true
	
	return false

func follow_mouse() -> void:
	# Skip if build mode is disabled
	if not build_mode:
		return
		
	# Get the mouse position in the viewport
	var mouse_pos = get_viewport().get_mouse_position()
	
	# Get the camera
	var camera = get_viewport().get_camera_3d()
	if not camera:
		return
	
	# Create a ray from the camera position through the mouse position
	var from = camera.project_ray_origin(mouse_pos)
	var to = from + camera.project_ray_normal(mouse_pos) * 1000
	
	# Cast a ray to find the ground plane
	var space_state = get_world_3d().direct_space_state
	var query = PhysicsRayQueryParameters3D.create(from, to)
	query.collision_mask = 0xFFFFFFFF  # All bits set
	var result = space_state.intersect_ray(query)
	
	# If we hit something, move the building
	if result and "position" in result:
		var hit_position = result.position
		
		# Apply the ground offset to raise it slightly (houses only)
		if current_building_index == 0:  # House
			hit_position.y += ground_offset
		elif current_building_index == 1:  # Cherry blossom
			# No extra height offset for trees
			pass
		
		global_position = hit_position
		
		# Make sure the scale is correct based on building type
		if model:
			if current_building_index == 0:  # House
				if not model.scale.is_equal_approx(Vector3(0.7, 0.7, 0.7)):
					model.scale = Vector3(0.7, 0.7, 0.7)
			elif current_building_index == 1:  # Cherry blossom
				if not model.scale.is_equal_approx(Vector3(0.08, 0.08, 0.08)):
					model.scale = Vector3(0.08, 0.08, 0.08)
					
func find_building_under_cursor() -> Node3D:
	# Get the mouse position in the viewport
	var mouse_pos = get_viewport().get_mouse_position()
	
	# Get the camera
	var camera = get_viewport().get_camera_3d()
	if not camera:
		return null
	
	# Create a ray from the camera position through the mouse position
	var from = camera.project_ray_origin(mouse_pos)
	var to = from + camera.project_ray_normal(mouse_pos) * 1000
	
	# Create an AABB for the current preview position (cursor position)
	var cursor_aabb = AABB(global_position - Vector3(0.5, 0.5, 0.5), Vector3(1, 1, 1))
	
	# Check if the ray intersects with any placed buildings
	for building in placed_buildings:
		if building == null or not is_instance_valid(building):
			continue
			
		var building_aabb = get_node_aabb(building)
		
		# Check if the ray passes through the building's AABB
		var intersection = ray_intersects_aabb(from, to, building_aabb)
		if intersection:
			return building
	
	return null

func ray_intersects_aabb(ray_origin: Vector3, ray_end: Vector3, aabb: AABB) -> bool:
	# Calculate ray direction and length
	var ray_direction = (ray_end - ray_origin).normalized()
	var ray_length = ray_origin.distance_to(ray_end)
	
	# Get AABB min and max points
	var aabb_min = aabb.position - aabb.size/2
	var aabb_max = aabb.position + aabb.size/2
	
	# Check if ray origin is inside AABB
	if aabb.has_point(ray_origin):
		return true
	
	# Calculate intersection with each of the 6 AABB planes
	var t_min = (aabb_min.x - ray_origin.x) / ray_direction.x if ray_direction.x != 0 else INF
	var t_max = (aabb_max.x - ray_origin.x) / ray_direction.x if ray_direction.x != 0 else -INF
	
	if t_min > t_max:
		var temp = t_min
		t_min = t_max
		t_max = temp
	
	var t_y_min = (aabb_min.y - ray_origin.y) / ray_direction.y if ray_direction.y != 0 else INF
	var t_y_max = (aabb_max.y - ray_origin.y) / ray_direction.y if ray_direction.y != 0 else -INF
	
	if t_y_min > t_y_max:
		var temp = t_y_min
		t_y_min = t_y_max
		t_y_max = temp
	
	if t_min > t_y_max or t_y_min > t_max:
		return false
	
	if t_y_min > t_min:
		t_min = t_y_min
	
	if t_y_max < t_max:
		t_max = t_y_max
	
	var t_z_min = (aabb_min.z - ray_origin.z) / ray_direction.z if ray_direction.z != 0 else INF
	var t_z_max = (aabb_max.z - ray_origin.z) / ray_direction.z if ray_direction.z != 0 else -INF
	
	if t_z_min > t_z_max:
		var temp = t_z_min
		t_z_min = t_z_max
		t_z_max = temp
	
	if t_min > t_z_max or t_z_min > t_max:
		return false
	
	if t_z_min > t_min:
		t_min = t_z_min
	
	if t_z_max < t_max:
		t_max = t_z_max
	
	# Check if intersection point is within ray length
	return t_min >= 0 && t_min <= ray_length

func delete_building_under_cursor():
	# Skip if build mode is disabled
	if not build_mode:
		return false
		
	var building = find_building_under_cursor()
	if building:
		print("Deleting building: " + str(building.name))
		
		# Remove from the placed_buildings array
		placed_buildings.erase(building)
		
		# Delete the building
		building.queue_free()
		
		# Request navigation rebake if ground_placements has the method
		if ground_placements and ground_placements.has_method("request_rebake"):
			ground_placements.request_rebake()
			
		return true
	
	return false

func clear_all_buildings():
	# Delete all placed buildings
	for building in placed_buildings:
		if building != null and is_instance_valid(building):
			building.queue_free()
	
	# Clear the array
	placed_buildings.clear()
	
	# Request navigation rebake if ground_placements has the method
	if ground_placements and ground_placements.has_method("request_rebake"):
		ground_placements.request_rebake()
		
	print("All buildings cleared")

func toggle_build_mode():
	build_mode = !build_mode
	
	# Toggle visibility of building preview
	if model:
		model.visible = build_mode
	
	if build_mode:
		print("Build mode enabled")
	else:
		print("Build mode disabled")

func _ready() -> void:
	# Check if required nodes exist
	if !model:
		print("ERROR: building_preview node not found")
		model = MeshInstance3D.new()
		add_child(model)
	
	if !area:
		print("ERROR: Area3D node not found")
	
	if !area_colshape:
		print("ERROR: CollisionShape3D node not found")
	
	if !ground_placements:
		print("ERROR: ground_placements node not found")
	
	if !game_state:
		print("ERROR: game_state node not found")
	
	# Make sure the model starts as red (invalid placement)
	model_red()
	
	# Scale down the model
	if model:
		model.scale = Vector3(0.7, 0.7, 0.7)
	
	# Load the first building model
	update_preview_model()
	
	print("Building preview ready")

func update_preview_model():
	# Clear existing model if any
	if model and model.mesh:
		model.mesh = null
	
	# Load the selected building
	var building_scene = load(all_buildings[current_building_index])
	if building_scene and model:
		var temp_instance = building_scene.instantiate()
		
		# Find the mesh in the loaded scene
		var mesh_instance = find_mesh_instance(temp_instance)
		if mesh_instance and mesh_instance.mesh:
			model.mesh = mesh_instance.mesh
		
		# Apply appropriate scale based on building type
		if current_building_index == 0:  # House
			model.scale = Vector3(0.7, 0.7, 0.7)
		elif current_building_index == 1:  # Cherry blossom
			model.scale = Vector3(1, 1, 1)  # Match smaller scale for preview
		
		# Clean up temporary instance
		temp_instance.queue_free()

func find_mesh_instance(node):
	# Recursively find the first MeshInstance3D
	if node is MeshInstance3D:
		return node
	
	for child in node.get_children():
		var result = find_mesh_instance(child)
		if result:
			return result
	
	return null

func _process(delta: float) -> void:
	# Only run these in build mode
	if build_mode:
		# Update the building's position to follow the mouse
		follow_mouse()
		
		# Check placement validity less frequently
		if Engine.get_frames_drawn() % 10 == 0:
			placement_check()
	
func _input(event: InputEvent) -> void:
	# Handle keyboard input 
	if event is InputEventKey and event.pressed:
		match event.keycode:
			KEY_1:
				# Delete building under cursor (only in build mode)
				if build_mode:
					if delete_building_under_cursor():
						print("Building deleted")
					else:
						print("No building to delete")
			KEY_2:
				# Save game state
				save_building_data()
				print("Game state saved")
			KEY_3:
				# Load game state
				load_building_data()
				print("Game state loaded")
			KEY_4:
				# Toggle build mode
				toggle_build_mode()
			KEY_5:
				# Select first building type (house)
				change_building_type(0)
			KEY_6:
				# Select second building type (cherry blossom)
				change_building_type(1)
	
	# Only process these inputs in build mode
	if build_mode:
		# Handle mouse button for placement
		if event is InputEventMouseButton:
			if event.button_index == MOUSE_BUTTON_LEFT and event.pressed:
				var can_place = placement_check()
				if can_place:
					place_building()
					print("Building placed successfully!")
				else:
					print("Cannot place building here")
			
			# Handle mouse wheel for rotation
			elif event.button_index == MOUSE_BUTTON_WHEEL_UP and event.pressed:
				# Rotate clockwise around Y axis
				rotate_y(deg_to_rad(-rotation_speed))
				
			elif event.button_index == MOUSE_BUTTON_WHEEL_DOWN and event.pressed:
				# Rotate counterclockwise around Y axis
				rotate_y(deg_to_rad(rotation_speed))
func place_building() -> void:
	# Get the current position
	var placement_position = global_position
	
	# Get ground top position
	var ground_y = ground_placements.global_position.y
	var ground_top = ground_y + (ground_placements.scale.y / 2) if ground_placements is CSGBox3D else ground_y
	
	# Set scale and height based on building type
	var building_scale = Vector3(0.7, 0.7, 0.7)  # Default scale
	
	if current_building_index == 0:  # House
		# Houses raised above ground
		placement_position.y = ground_top + placement_offset
	elif current_building_index == 1:  # Cherry blossom tree
		# Make tree smaller and place it slightly into the ground
		building_scale = Vector3(0.02, 0.02, 0.02)  # 40% of original size
		placement_position.y = ground_top +0.1 # Slightly into the ground
	
	# Load the building from our list
	var building_scene = load(all_buildings[current_building_index])
	if building_scene:
		var building = building_scene.instantiate()
		
		# Create a new transform that combines our rotation with the new position
		var new_transform = Transform3D(
			global_transform.basis,  # This contains the rotation
			placement_position       # This is our new position
		)
		
		# Apply the transform to the new building
		building.global_transform = new_transform
		
		# Apply the appropriate scale
		building.scale = building_scale
		
		# Add the building to the scene
		get_parent().add_child(building)
		
		# Keep track of placed buildings
		placed_buildings.append(building)
		
		# Store the building type index with the building
		if building.get_meta_list().find("building_type") == -1:
			building.set_meta("building_type", current_building_index)
		
		# Store the scale with the building
		building.set_meta("custom_scale", building_scale)
		
		# Request navigation rebake if ground_placements has the method
		if ground_placements and ground_placements.has_method("request_rebake"):
			ground_placements.request_rebake()
		
		print("Building placed at height: " + str(placement_position.y) + 
			  " with rotation: " + str(rotation_degrees) + 
			  " (Type: " + str(current_building_index) + ")")
func save_building_data():
	# Call the save function on the game_state node
	if game_state and game_state.has_method("save_game"):
		# Collect data about all buildings
		var building_data = []
		for building in placed_buildings:
			if building != null and is_instance_valid(building):
				# Get the building type (either from metadata or default to 0)
				var building_type = 0
				if building.get_meta_list().find("building_type") != -1:
					building_type = building.get_meta("building_type")
				
				building_data.append({
					"position": {
						"x": building.global_position.x,
						"y": building.global_position.y,
						"z": building.global_position.z
					},
					"rotation": {
						"x": building.rotation.x,
						"y": building.rotation.y,
						"z": building.rotation.z
					},
					"scale": {
						"x": building.scale.x,
						"y": building.scale.y,
						"z": building.scale.z
					},
					"type": building_type  # Index in all_buildings array
				})
		
		# Pass the data to game_state
		game_state.save_game(building_data)

func load_building_data():
	# Call the load function on the game_state node
	if game_state and game_state.has_method("load_game"):
		var building_data = game_state.load_game()
		
		if building_data:
			# Clear existing buildings
			clear_all_buildings()
			
			# Spawn new buildings based on loaded data
			for data in building_data:
				# Get the building type
				var building_type_index = data.type if "type" in data else 0
				if building_type_index < 0 or building_type_index >= all_buildings.size():
					building_type_index = 0
				
				# Load the building scene
				var building_scene = load(all_buildings[building_type_index])
				if building_scene:
					var building = building_scene.instantiate()
					
					# Set position
					var position = Vector3(
						data.position.x,
						data.position.y,
						data.position.z
					)
					
					# Set rotation
					var rotation_vec = Vector3(
						data.rotation.x,
						data.rotation.y,
						data.rotation.z
					)
					
					# Set scale
					var scale_vec = Vector3(
						data.scale.x,
						data.scale.y,
						data.scale.z
					)
					
					# Apply transform
					building.global_position = position
					building.rotation = rotation_vec
					building.scale = scale_vec
					
					# Store the building type index with the building
					building.set_meta("building_type", building_type_index)
					
					# Add to scene
					get_parent().add_child(building)
					
					# Track the building
					placed_buildings.append(building)
			
			# Request navigation rebake after loading all buildings
			if ground_placements and ground_placements.has_method("request_rebake"):
				ground_placements.request_rebake()
				
			print("Loaded " + str(building_data.size()) + " buildings")
