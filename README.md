# Feedly Asset Sync to custom lists
This repository aims to make it easier to upload assets to feedly custom lists.
For this purpose, this is designed as a monorepo with three different programs.
All of them have the flaw, that every asset uploaded is from the type "customKeyword". So the script doesn't generally look if a similar asset exists already as a builtin type in feedly. This may hinder Feedlys capabilities to do the most with your data, so use with care!

- **feedly_asset_sync_script**  
A Python script which is able to fetch data from jira Assets with a custom AQL query and uploads it in correct batch sizes of 50 to Feedly custom lists
- **feedly_asset_uploader_cli**  
An executable file written in Golang which fetches the data from a premade csv file and uploads it to feedly. It has no control built in in regards to the batchsize (50 items per list). It is a command line program which has to be executed in a shell.
- **feedly_asset_uploader_gui**  
An executable file written in Golang with Wails and Vue. This program has the same limitations as the cli version, but it is built to be more appealing to users. It can be executed as a standalone executable, and provides an interface over an embedded Webview. It takes the upload URL and the API Key and after saving, one can upload a csv file directly into Feedly custom lists.

## Usage instructions
### feedly_asset_sync_script
1. Rename the config.example.json to config.json and fill the fields with the correct information
2. Install dependencies:
   - Create a virtual environment and install the dependencies:
        ```bash
        python3 -m venv .venv
        source .venv/bin/activate
        pip install -r requirements.txt
        ```
    - It is to be noted, that the only requirement in this case is the requests library. If it is already available in your environment, then this isn't necessary.
3. Start the script with the config.json file in the same directory. You can run it via cron to have the synchronization up to date.
### feedly_asset_uploader_cli
1. Given that Golang is already installed, you do not need to do have a specific setup since the program uses only standard libraries.
2. Run the program with `go run main.go` or build an executable file with `go build -o app` which will create a build system dependent binary called `app`.
3. Before running the program make sure you configured the app correctly through the config.json file in the same directory as the app.
### feedly_asset_uploader_gui
1. This project is built with wails, you can find more information here about installing it: https://wails.io/docs/gettingstarted/installation
2. Because Vue is used, you also have to make sure that npm is available. The dependencies can be found in the package.json file.
3. The development server can be started with `wails dev` and a production ready executable can be build with `wails build`.
Follow the wails documentation for more information about creating an installer with nsis or compressing the executable file with upx.