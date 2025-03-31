@icon("res://icons/hammericon.png")
class_name LevelGenerator extends Node

static func generate_nodes(json: Variant) -> Node:
	if json is String:
		json = JSON.parse_string(json)
	if json is not Array:
		return null
	if !json.all(checkItem):
		return null
	var level = GeneratedLevel.new()
	level.name = "GeneratedLevel"
	level.regions = json
	for item in json:
		var region = Node2D.new()
		var byte_pool = Marshalls.base64_to_raw(item["regionImage"])
		var img = Image.new()
		img.load_png_from_buffer(byte_pool)
		var sprite = Sprite2D.new()
		sprite.name = "Sprite"
		sprite.centered = false
		sprite.texture = ImageTexture.create_from_image(img)
		region.add_child(sprite)
		var collision = CollisionPolygon2D.new()
		var shape = item["shape"] as Array
		var vectorshape = shape.map(func(v: Dictionary): return Vector2(v["x"], v["y"]))
		collision.polygon = vectorshape
		var col = StaticBody2D.new()
		col.name = "Collider"
		col.add_child(collision)
		region.add_child(col)
		region.position = Vector2(item["cornerX"], item["cornerY"])
		var color = item["regionColorString"]
		match color:
			"Red":
				col.add_to_group("Red")
				sprite.add_to_group("Red")
			"Green":
				col.add_to_group("Green")
				sprite.add_to_group("Green")
			"Blue":
				col.add_to_group("Blue")
				sprite.add_to_group("Blue")
			"Black":
				col.add_to_group("Black")
				sprite.add_to_group("DarkModeInvertColors")
		level.add_child(region)
	return level

static func checkItem(item: Variant) -> bool:
	return item is Dictionary and item.get("regionImage") is String and item.get("shape") is Array \
		and item["shape"].all(func(m): return m is Dictionary and m.has_all(["x", "y"])) and item.has_all(["cornerX", "cornerY"])

class GeneratedLevel extends Node:
	var regions: Array
