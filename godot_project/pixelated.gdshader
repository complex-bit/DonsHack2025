shader_type canvas_item;

uniform sampler2D SCREEN_TEXTURE : hint_screen_texture, filter_linear_mipmap;
uniform int pixel_size : hint_range(1, 64) = 4; // Pixel size

void fragment() {
    // Get the size of the ColorRect
    vec2 rect_size = 1.0 / SCREEN_PIXEL_SIZE;
    
    // Calculate pixelated UV coordinates relative to the ColorRect
    vec2 pixelated_uv = floor(UV * rect_size / float(pixel_size)) * float(pixel_size) / rect_size;
    
    // Sample the screen texture at the pixelated coordinates
    COLOR = texture(SCREEN_TEXTURE, pixelated_uv);
}