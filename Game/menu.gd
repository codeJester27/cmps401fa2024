extends Control

func _ready():
	if OS.has_feature("web"):
		$MarginContainer/VBoxContainer/Exit.hide()
	if RenderingServer.get_default_clear_color() == Color(0, 0, 0, 1):
		$OptionsWindow/DarkMode.set_pressed_no_signal(true)
	$OptionsWindow/HatToggle.set_pressed_no_signal(ProjectSettings.get_setting("application/boardwalk/hat_enabled"))
	$OptionsWindow/ColorblindMode.set_pressed_no_signal(ProjectSettings.get_setting("rendering/environment/defaults/color_blind_mode"))
	$MarginContainer/VBoxContainer/Start.call_deferred("grab_focus")
	
func _on_start_pressed():
	get_tree().change_scene_to_file("res://start_menu.tscn")


func _on_exit_pressed():
	get_tree().quit()

func _on_options_pressed():
	$OptionsWindow.show()
	
func _on_options_close_pressed():
	$OptionsWindow.hide()
