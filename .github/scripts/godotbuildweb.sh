echo Downloading Godot...
curl -o godot.zip -L 'https://github.com/godotengine/godot/releases/download/4.3-stable/Godot_v4.3-stable_linux.x86_64.zip'
echo Downloading export templates...
curl -o templates.zip -L 'https://github.com/godotengine/godot/releases/download/4.3-stable/Godot_v4.3-stable_export_templates.tpz'
echo Installing unzip...
sudo apt install unzip
echo Unzipping Godot...
unzip -p godot.zip Godot_v*-stable_linux.x86_64 > godot.x86_64
echo Unzipping export templates...
unzip templates.zip templates/web_nothreads_debug.zip templates/web_nothreads_release.zip
echo Installing export templates...
mkdir -p ~/.local/share/godot/export_templates/4.3.stable
cp ./templates/web_nothreads_debug.zip ~/.local/share/godot/export_templates/4.3.stable/web_nothreads_debug.zip
cp ./templates/web_nothreads_debug.zip ~/.local/share/godot/export_templates/4.3.stable/web_nothreads_release.zip
chmod +xr godot.x86_64
echo Exporting project...
mkdir exported-game
./godot.x86_64 --export-release Web ../WebApp/exported-game/index.html --path ../Game --headless
echo Godot project exported.