shader_type spatial;
render_mode blend_mix, diffuse_toon,specular_disabled,shadows_disabled,ambient_light_disabled;
instance uniform vec4 instance_color_01 : source_color; 

void fragment() {
	ALBEDO = instance_color_01.rgb; 
	ALPHA = instance_color_01.a; 
}

//void light() {
//	// Called for every pixel for every light affecting the material.
//	// Uncomment to replace the default light processing function with this one.
//}
