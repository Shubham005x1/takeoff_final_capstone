.PHONY: build

build:
	# Create a zip file named "CreateGrocery.zip" using PowerShell
	PowerShell Compress-Archive -Path utils, validations,go.mod,go.sum, common, cloudfunctions/CreateGrocery.go -DestinationPath CreateGrocery.zip
