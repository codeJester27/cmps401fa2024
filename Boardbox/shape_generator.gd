@icon("res://icons/hammericon.png")
class_name ShapeGenerator extends Node

func generate_nodes(json_string: String) -> Node:
	var json = JSON.parse_string(json_string)
	if json is not Array:
		return null
	if !json.all(checkItem):
		return null
	var level = Node.new()
	for item in json:
		
		var byte_pool = Marshalls.base64_to_raw(item["regionImage"])
		var img = Image.new()
		img.load_png_from_buffer(byte_pool)
		
		var region = RigidBody2D.new()
		region.center_of_mass_mode = RigidBody2D.CENTER_OF_MASS_MODE_CUSTOM
		region.center_of_mass = img.get_size() / 2
		region.continuous_cd = RigidBody2D.CCD_MODE_CAST_RAY
		
		var sprite = Sprite2D.new()
		sprite.centered = false
		sprite.texture = ImageTexture.create_from_image(img)
		region.add_child(sprite)
		
		var collision = CollisionPolygon2D.new()
		var shape = item["shape"] as Array
		var vectorshape = shape.map(func(v: Dictionary): return Vector2(v["x"], v["y"]))
		
		collision.polygon = vectorshape
		region.add_child(collision)
		region.position = Vector2(item["cornerX"], item["cornerY"])
		level.add_child(region)
	return level

func checkItem(item: Variant) -> bool:
	return item is Dictionary and item.get("regionImage") is String and item.get("shape") is Array \
		and item["shape"].all(func(m): return m is Dictionary and m.has_all(["x", "y"])) and item.has_all(["cornerX", "cornerY"])
