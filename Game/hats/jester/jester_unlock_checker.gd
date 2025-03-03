class_name JesterUnlockChecker extends UnlockChecker

static func get_hat_id() -> String:
	return "jester"

func _ready():
	var valid = JavaScriptBridge.eval("""
const url = new URL(window.location);

const today = new Date();
today.setHours(0, 0, 0, 0);
const todayHash = today.toISOString().split("").reduce((a, b) => {
  a = ((a << 5) - a) + b.charCodeAt(0);
  return a & a;
}, 0);

url.searchParams.get("unlock") === `${todayHash}`;
""")
	if valid:
		Unlocks.unlock_hat(get_hat_id())
		JavaScriptBridge.eval("""
const url = new URL(window.location);
url.search = "";
history.replaceState(null, undefined, url);
""")
