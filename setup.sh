#!/bin/bash

# Check if the user has permission to write to /usr/local/bin
if [ "$(id -u)" != "0" ]; then
   echo "Error: This command must be run as root." 1>&2
   exit 1
fi

echo "🚀 Starting installation of NetoKeep..."
# 2. Copy binary file to system directory
# Note: ./nk refers to the file in the extracted temporary directory
cp -f ./nk /usr/local/bin/nk
cp -f ./nks /usr/local/bin/nks

# 3. Set execute permissions
chmod 755 /usr/local/bin/nk
chmod 755 /usr/local/bin/nks

echo "✅ Installation completed successfully!"
echo "👉 Now use 'nk' and 'nks' anywhere in your shell."
