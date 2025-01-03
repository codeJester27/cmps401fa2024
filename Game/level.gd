extends Node

var base_url = ProjectSettings.get_setting("application/boardwalk/web_server_url")

@onready var level_generator = $LevelGenerator
@onready var loading_indicator = $LoadingIndicator
@onready var multiplayer_timer = $MultiplayerTimer
@onready var multiplayer_controller = $MultiplayerController

var current_campaign_level: String = ""

func _ready():
	$"QuitMenu/QuitWindow/volume-slider".set_value_no_signal(100)
	if RenderingServer.get_default_clear_color() == Color(0, 0, 0, 1):
		$QuitMenu/QuitWindow/DarkCheck.set_pressed_no_signal(true)
	$QuitMenu/QuitWindow/ColorCheck.set_pressed_no_signal(ProjectSettings.get_setting("rendering/environment/defaults/color_blind_mode"))

func create_level(img: Image, options: Dictionary):
	loading_indicator.show()
	loading_indicator.set_text("Uploading Level...")
	if RenderingServer.get_default_clear_color() == Color(0, 0, 0, 1):
		loading_indicator.set_text_color(Color(1, 1, 1, 1))
	var buffer = img.save_png_to_buffer()
	var request = FileUploader.upload_buffer(base_url + "/api/build-level", buffer, "image.png", HTTPClient.METHOD_POST, "image", options)
	request.request_completed.connect(_on_response_received)

func load_level(level_data: Variant):
	if level_data is PackedByteArray:
		level_data = level_data.get_string_from_utf8()
	
	if level_data is not String:
		show_error("Could not load level.")
		return
	
	var json = JSON.parse_string(level_data)
	
	if json is not Dictionary or json.get("startPos") is not Dictionary \
		or json.get("endPos") is not Dictionary or json.get("regions") is not Array:
		show_error("Could not load level.")
		return
	
	var generated_level = level_generator.generate_nodes(json["regions"])
	if generated_level == null:
		show_error("Could not load level.")
		return
	
	add_child(generated_level)
	var start_pos = json["startPos"]
	var end_pos = json["endPos"]
	start_game(str(hash(level_data)), Vector2(start_pos["x"], start_pos["y"]), Vector2(end_pos["x"], end_pos["y"]))

func _on_response_received(result: int, response_code: int, headers: PackedStringArray, body: PackedByteArray):
	if response_code != HTTPClient.RESPONSE_OK:
		show_error(body, response_code)
		return
	
	var level_data = body.get_string_from_utf8()
	var generated_level = level_generator.generate_nodes(level_data)
	if generated_level == null:
		show_error("Could not generate level with server response.")
		return
	add_child(generated_level)
	start_game(str(hash(level_data)))

func start_game(multiplayer_id: String, start_pos: Vector2 = Vector2.ZERO, end_pos: Vector2 = Vector2.ZERO):
	if RenderingServer.get_default_clear_color() == Color(0, 0, 0, 1):
		$QuitButton.material = ShaderMaterial.new()
		$QuitButton.material.shader = load("res://color_invert.gdshader")
	var player = add_player()
	multiplayer_controller.try_connect(multiplayer_id)
	loading_indicator.hide()
	if start_pos == Vector2.ZERO and end_pos == Vector2.ZERO:
		get_tree().paused = true
		$StartEndSelection/StartSelect.disabled = false
		$StartEndSelection/StartSelect.show()
	elif start_pos != Vector2.ZERO and end_pos != Vector2.ZERO:
		player.initial_position = start_pos
		player.position = player.initial_position
		var goal = $Goal
		goal.position = end_pos
		goal.show()
		$TouchScreenControls.show()
	else:
		assert(false, "make sure to set either both start and end positions or neither of them")

func show_error(body: Variant, error_code: int = 0):
	var error_dialog = ErrorDialog.new()
	if error_code != 0:
		error_dialog.set_text_to_error_message(body, error_code)
	else:
		error_dialog.set_text_to_error_message(body)
	error_dialog.show()
	add_child(error_dialog)
	error_dialog.confirmed.connect(go_back)
	error_dialog.canceled.connect(go_back)
	

func add_player() -> Player:
	var player = preload("res://player.tscn").instantiate()
	if RenderingServer.get_default_clear_color() == Color(0, 0, 0, 1):
		player.get_node("AnimatedSprite2D").material = ShaderMaterial.new()
		player.get_node("AnimatedSprite2D").material.shader = load("res://color_invert.gdshader")
	add_child(player)
	return player
	
func _on_quit_button_pressed():
	get_node("./QuitMenu/QuitWindow").show()
	get_tree().paused = true

func _on_back_button_pressed():
	get_tree().paused = false
	go_back()

func go_back():
	get_tree().change_scene_to_file("res://start_menu.tscn")

func _on_exit_to_main_menu_button_pressed():
	get_tree().change_scene_to_file("res://main.tscn")
	

func _set_player_start():
	var player = $Player
	player.initial_position = get_viewport().get_mouse_position()
	player.position = player.initial_position
	$StartEndSelection/StartSelect.disabled = true
	$StartEndSelection/StartSelect.hide()
	$StartEndSelection/EndSelect.disabled = false
	$StartEndSelection/EndSelect.show()

func _set_goal_position():
	var goal = $Goal
	goal.position = get_viewport().get_mouse_position()
	$StartEndSelection/EndSelect.disabled = true
	$StartEndSelection/EndSelect.hide()
	$TouchScreenControls.show()
	$Goal.show()
	get_tree().paused = false
	
func _goal_reached(player: Node2D):
	if current_campaign_level != "":
		var tree = get_tree()
		var currlevel = CampagignLevels.levels.find(current_campaign_level)
		if currlevel + 1 == len(CampagignLevels.levels):
			player.set_physics_process(false)
			$VictoryScreen.show()
			%Restart.call_deferred("grab_focus")
		else:
			var nextlevel = CampagignLevels.levels[currlevel + 1]
			var next_level_node = preload("res://level.tscn").instantiate()
			next_level_node.current_campaign_level = nextlevel
			add_sibling(next_level_node)
			next_level_node.load_level(FileAccess.get_file_as_string(nextlevel))
			get_tree().set_deferred("current_scene", next_level_node)
			queue_free()
	else:
		player.set_physics_process(false)
		$VictoryScreen.show()
		%Restart.call_deferred("grab_focus")
	
func _on_audio_stream_player_finished():
	$AudioStreamPlayer.play()

func _on_restart_button_pressed():
	$VictoryScreen.hide()
	$Goal.hide()
	
	loading_indicator.hide()
	get_tree().paused = true
	$StartEndSelection/StartSelect.disabled = false
	$StartEndSelection/StartSelect.show()
	var player = $Player
	player.set_physics_process(true)

func _on_multiplayer_timer_timeout():
	var player: CharacterBody2D = get_node_or_null("Player")
	if player != null:
		var sprite = player.get_node("AnimatedSprite2D") as AnimatedSprite2D
		$MultiplayerController.send_player_info(Preferences.get_player_name(), sprite.animation, sprite.frame, player.position, sprite.flip_h)

func _on_quit_window_close_requested():
	get_tree().paused = false
	$QuitMenu/QuitWindow.hide()

func _on_volumeslider_value_changed(value: float):
	$AudioStreamPlayer.set_volume_db(value - 100)

func _on_color_check_toggled(toggled: bool):
	var level = get_node("GeneratedLevel")
	ProjectSettings.set_setting("rendering/environment/defaults/color_blind_mode", toggled)
	if toggled:
		for child in level.get_children():
			if child.get_node("Collider").is_in_group("Red"):
				child.get_node("Sprite").material = ShaderMaterial.new()
				child.get_node("Sprite").material.shader = load("res://colorblind_red.gdshader")
				child.get_node("Sprite").material.set("shader_parameter/tile_size", 1)
				child.get_node("Sprite").material.set("shader_parameter/pattern", load("res://red_cb.png"))
			if child.get_node("Collider").is_in_group("Green"):
				child.get_node("Sprite").material = ShaderMaterial.new()
				child.get_node("Sprite").material.shader = load("res://colorblind_green.gdshader")
				child.get_node("Sprite").material.set("shader_parameter/tile_size", 1)
				child.get_node("Sprite").material.set("shader_parameter/pattern", load("res://green_cb.png"))
			if child.get_node("Collider").is_in_group("Blue"):
				child.get_node("Sprite").material = ShaderMaterial.new()
				child.get_node("Sprite").material.shader = load("res://colorblind_blue.gdshader")
				child.get_node("Sprite").material.set("shader_parameter/tile_size", 1)
				child.get_node("Sprite").material.set("shader_parameter/pattern", load("res://blue_cb.png"))
	else:
		for child in level.get_children():
			if child.get_node("Collider").is_in_group("Red"):
				child.get_node("Sprite").set_material(null)
			if child.get_node("Collider").is_in_group("Green"):
				child.get_node("Sprite").set_material(null)
			if child.get_node("Collider").is_in_group("Blue"):
				child.get_node("Sprite").set_material(null)


func _on_dark_check_toggled(toggled: bool):
	if toggled:
		RenderingServer.set_default_clear_color(Color(0, 0, 0, 1))
		$Player.get_node("AnimatedSprite2D").material = ShaderMaterial.new()
		$Player.get_node("AnimatedSprite2D").material.shader = load("res://color_invert.gdshader")
		$QuitButton.material = ShaderMaterial.new()
		$QuitButton.material.shader = load("res://color_invert.gdshader")
		var level = get_node("GeneratedLevel")
		for child in level.get_children():
			if child.get_node("Collider").is_in_group("Black"):
				child.get_node("Sprite").material = ShaderMaterial.new()
				child.get_node("Sprite").material.shader = load("res://color_invert.gdshader")
	else:
		RenderingServer.set_default_clear_color(Color(1, 1, 1, 1))
		$Player.get_node("AnimatedSprite2D").set_material(null)
		$QuitButton.set_material(null)
		var level = get_node("GeneratedLevel")
		for child in level.get_children():
			if child.get_node("Collider").is_in_group("Black"):
				child.get_node("Sprite").set_material(null)

func save_level():
	var player = $Player as Player
	var goal = $Goal as Area2D
	var generated_level = $GeneratedLevel as LevelGenerator.GeneratedLevel
	var level_info = {
		"startPos": { "x": player.initial_position.x, "y": player.initial_position.y },
		"endPos": { "x": goal.position.x, "y": goal.position.y },
		"regions": generated_level.regions
	}
	var json = JSON.stringify(level_info)
	if OS.has_feature("web"):
		JavaScriptBridge.download_buffer(json.to_utf8_buffer(), "level.boardwalk", "application/json")
	else:
		var file := FileAccess.open("user://level.boardwalk", FileAccess.WRITE)
		file.store_string(json)
		file.close()
		OS.shell_show_in_file_manager(ProjectSettings.globalize_path("user://level.boardwalk"))
	

func _on_resume_button_pressed():
	get_tree().paused = false
	$QuitMenu/QuitWindow.hide()
