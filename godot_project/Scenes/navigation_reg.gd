extends NavigationRegion3D

# Signal that other scripts can connect to
signal nav_mesh_updated

func _ready():
	# Add to a group for easy reference
	add_to_group("navigation_mesh_instance")
	
	# Initial bake
	bake_navigation_mesh()
	print("Navigation mesh initially baked")

# Function that other scripts can call to trigger a rebake
func request_rebake():
	bake_navigation_mesh(true)
	print("Navigation mesh rebaked")
	emit_signal("nav_mesh_updated")
