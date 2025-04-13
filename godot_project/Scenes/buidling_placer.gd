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
var all_buildings = ["res://house_6.tscn", "res://buildings/cherry_blossom.tscn"]
var current_building_index = 0

# Define consistent scales for each building type (both preview and placement)
var building_scales = [
	Vector3(0.7, 0.7, 0.7),    # House scale
	Vector3(0.04, 0.04, 0.04)  # Cherry blossom scale (for placement)
]

# Make the preview larger for visibility while keeping the placement scale small
var preview_scales = [
	Vector3(0.7, 0.7, 0.7),    # House preview scale (same as placement)
	Vector3(0.03, 0.03, 0.03)  # Cherry blossom preview scale (larger for visibility)
]

# Track all placed buildings
var placed_buildings = []

# Build mode toggle
var build_mode = true

# Store the real hit position for placement (without preview offsets)
var true_hit_position = Vector3()

func model_red() -> void:
	if model:
		if current_building_index == 1:  # Cherry blossom tree
			_set_preview_color(model, Color(1.0, 0.0, 0.0, 1.0))  # Red with full opacity
		else:
			# For house, use the standard approach
			if model.get_class() == "MeshInstance3D":
				model.set("instance_shader_parameters/instance_color_01", Color("red"))

func model_blue() -> void:
	if model:
		if current_building_index == 1:  # Cherry blossom tree
			_set_preview_color(model, Color(0.0, 0.5, 1.0, 1.0))  # Blue with full opacity
		else:
			# For house, use the standard approach
			if model.get_class() == "MeshInstance3D":
				model.set("instance_shader_parameters/instance_color_01", Color("blue"))

# Helper function to set color on all preview materials
func _set_preview_color(node: Node, color: Color) -> void:
	if node is MeshInstance3D:
		for i in range(node.get_surface_override_material_count()):
			var mat = node.get_surface_override_material(i)
			if mat and mat is ShaderMaterial:
				mat.set_shader_parameter("instance_color_01", color)
	
	# Apply to all children recursively
	for child in node.get_children():
		_set_preview_color(child, color)
	
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
	
	# For trees we need to use a point slightly above the ground
	var check_position = global_position
	if current_building_index == 1:  # Cherry blossom
		# Use the true hit position for checking ground placement
		check_position = true_hit_position
	
	# Define the four corner points
	var points_to_check: Array = [
		Vector3(check_position.x + bounds.x, check_position.y - bounds.y, check_position.z + bounds.z),
		Vector3(check_position.x + bounds.x, check_position.y - bounds.y, check_position.z - bounds.z),
		Vector3(check_position.x - bounds.x, check_position.y - bounds.y, check_position.z - bounds.z),
		Vector3(check_position.x - bounds.x, check_position.y - bounds.y, check_position.z + bounds.z)
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
		
		# Store the true hit position for placement (don't modify this for preview offset)
		true_hit_position = hit_position
		
		# Apply appropriate height offset based on building type
		if current_building_index == 0:  # House
			hit_position.y += ground_offset
		elif current_building_index == 1:  # Cherry blossom
			# Slightly raise preview above ground
			hit_position.y -= 0.3
		
		global_position = hit_position
		
		# Update the model scale based on building type
		if model:
			# Use the defined preview scale for the current building type
			var current_scale = preview_scales[current_building_index]
			if not model.scale.is_equal_approx(current_scale):
				model.scale = current_scale
					
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
		model = Node3D.new()  # Changed to Node3D to act as a container
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
	
	# Load the first building model
	update_preview_model()
	
	print("Building preview ready")

# Modified update_preview_model function to preserve leaf transparency
func update_preview_model():
	# Clear existing model if any
	if model:
		# Remove any existing children
		for child in model.get_children():
			child.queue_free()
		
		if model.mesh:
			model.mesh = null
	
	# Load the selected building
	var building_scene = load(all_buildings[current_building_index])
	if building_scene:
		var temp_instance = building_scene.instantiate()
		
		if current_building_index == 1:  # Cherry blossom - special handling for leaves
			# For cherry blossom, we'll use the whole scene instance as preview
			# instead of just the mesh
			
			# Make a clone of the tree for preview
			var tree_preview = building_scene.instantiate()
			
			# Add it as a child of our model node
			model.add_child(tree_preview)
			
			# Apply the preview scale to our model
			model.scale = preview_scales[current_building_index]
			
			# Make the tree preview itself have scale 1,1,1 since the parent has the scale
			tree_preview.scale = Vector3(1, 1, 1)
			
			# Instead of replacing materials, duplicate them and modify the duplicates
			_prepare_preview_materials(tree_preview)
			
			# Set initial color (red by default)
			model_red()
		else:
			# For other models (like houses), use the old approach
			var mesh_instance = find_mesh_instance(temp_instance)
			if mesh_instance and mesh_instance.mesh:
				model.mesh = mesh_instance.mesh
				
				# Apply appropriate scale based on building type
				model.scale = preview_scales[current_building_index]
		
		# Clean up temporary instance
		temp_instance.queue_free()

# New function to prepare the materials for preview without losing original properties
func _prepare_preview_materials(node: Node) -> void:
	if node is MeshInstance3D and node.mesh:
		for surface_idx in range(node.mesh.get_surface_count()):
			var original_material = node.mesh.surface_get_material(surface_idx)
			if original_material:
				# Try to detect if this is a leaf material
				var is_leaf = false
				if original_material is StandardMaterial3D:
					# Leaves typically have alpha transparency or alpha scissor enabled
					is_leaf = original_material.transparency == StandardMaterial3D.TRANSPARENCY_ALPHA || original_material.transparency == StandardMaterial3D.TRANSPARENCY_ALPHA_SCISSOR
				
				# Create different shader based on material type
				var preview_shader = Shader.new()
				
				if is_leaf:
					# Enhanced shader for leaves that preserves their appearance
					preview_shader.code = """
					shader_type spatial;
					render_mode blend_mix, cull_disabled, depth_prepass_alpha, unshaded;
					
					uniform vec4 instance_color_01 : source_color;
					uniform sampler2D texture_albedo : source_color, filter_linear_mipmap, repeat_enable;
					
					void fragment() {
						vec4 albedo_tex = texture(texture_albedo, UV);
						
						// Preserve the original alpha from the texture
						float alpha = albedo_tex.a;
						
						// Blend the instance color with the texture's alpha
						ALBEDO = instance_color_01.rgb;
						ALPHA = alpha * instance_color_01.a;
					}
					"""
					
					# Create a new material with our shader
					var preview_material = ShaderMaterial.new()
					preview_material.shader = preview_shader
					
					# Try to get the albedo texture from the original material
					if original_material is StandardMaterial3D and original_material.albedo_texture:
						preview_material.set_shader_parameter("texture_albedo", original_material.albedo_texture)
					
					# Set initial color (will be updated by model_red/blue)
					preview_material.set_shader_parameter("instance_color_01", Color(1.0, 0.0, 0.0, 0.7))
					
					# Apply the preview material to this surface
					node.set_surface_override_material(surface_idx, preview_material)
				else:
					# Standard preview shader for non-leaf parts
					preview_shader.code = """
					shader_type spatial;
					render_mode blend_mix, diffuse_toon, specular_disabled, shadows_disabled, ambient_light_disabled;
					
					uniform vec4 instance_color_01 : source_color; 
					
					void fragment() {
						ALBEDO = instance_color_01.rgb; 
						ALPHA = instance_color_01.a; 
					}
					"""
					
					# Create a new material with our shader
					var preview_material = ShaderMaterial.new()
					preview_material.shader = preview_shader
					preview_material.set_shader_parameter("instance_color_01", Color(1.0, 0.0, 0.0, 1.0))
					
					# Apply the preview material to this surface
					node.set_surface_override_material(surface_idx, preview_material)
	
	# Process all children recursively
	for child in node.get_children():
		_prepare_preview_materials(child)

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
	# Get ground top position
	var ground_y = ground_placements.global_position.y
	var ground_top = ground_y + (ground_placements.scale.y / 2) if ground_placements is CSGBox3D else ground_y
	
	# Start with the true position from the raycast
	var placement_position = true_hit_position
	
	# Adjust height based on building type
	if current_building_index == 0:  # House
		# Houses raised above ground
		placement_position.y = ground_top + placement_offset
	elif current_building_index == 1:  # Cherry blossom tree
		# Place tree directly on the ground
		placement_position.y = ground_top + 0.1  # Small offset to avoid z-fighting
	
	# Handle preview visibility - without await
	if model and model.get_children().size() > 0:
		for child in model.get_children():
			if child != null and is_instance_valid(child):
				child.visible = false
		
		# Use a timer to show preview again after a small delay
		var timer = Timer.new()
		add_child(timer)
		timer.wait_time = 0.1
		timer.one_shot = true
		timer.timeout.connect(func():
			if model:
				for child in model.get_children():
					if child != null and is_instance_valid(child):
						child.visible = true
			timer.queue_free()
		)
		timer.start()
	
	# Load the building from our list
	var building_scene = load(all_buildings[current_building_index])
	if building_scene:
		var new_building = building_scene.instantiate()
		
		# Create a new transform that combines our rotation with the new position
		var new_transform = Transform3D(
			global_transform.basis,  # This contains the rotation
			placement_position       # This is our new position
		)
		
		# Apply the transform to the new building
		new_building.global_transform = new_transform
		
		# Apply the appropriate scale from our placement scale array
		new_building.scale = building_scales[current_building_index]
		
		# Add the building to the scene
		get_parent().add_child(new_building)
		
		# Keep track of placed buildings
		placed_buildings.append(new_building)
		
		# Store the building type index with the building
		if new_building.get_meta_list().find("building_type") == -1:
			new_building.set_meta("building_type", current_building_index)
		
		# Store the scale with the building
		new_building.set_meta("custom_scale", building_scales[current_building_index])
		
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
					var new_building = building_scene.instantiate()
					
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
					new_building.global_position = position
					new_building.rotation = rotation_vec
					new_building.scale = scale_vec
					
					# Store the building type index with the building
					new_building.set_meta("building_type", building_type_index)
					
					# Add to scene
					get_parent().add_child(new_building)
					
					# Track the building
					placed_buildings.append(new_building)
			
			# Request navigation rebake after loading
