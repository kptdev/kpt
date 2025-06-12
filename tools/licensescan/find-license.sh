#!/bin/bash

# This script reads a URL from the first line of a specified file,
# fetches the web page content, extracts the license information from that page,
# and then updates the 'license: TODO' line in the file with the found license.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
# Ensure 'curl' and 'grep' (with Perl-compatible regex support, -P) are installed
# on your system.

# --- Input Validation ---
# Check if a filename argument is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <filename>" >&2
  echo "Example: $0 my_package_info.txt" >&2
  exit 1
fi

FILE="$1"

# Check if the input file exists
if [ ! -f "$FILE" ]; then
  echo "Error: File '$FILE' not found." >&2
  exit 1
fi

# --- Extract URL from file ---
# Read the first line of the file and remove the leading '# ' to get the URL.
# head -n 1: reads only the first line.
# sed 's/^# //' : removes the '# ' prefix.
URL=$(head -n 1 "$FILE" | sed 's/^# //')

# Check if a URL was successfully extracted
if [ -z "$URL" ]; then
  echo "Error: Could not extract URL from the first line of '$FILE'." >&2
  echo "Ensure the first line starts with '# ' followed by the URL." >&2
  exit 1
fi

echo "Found URL in '$FILE': $URL"

# --- Fetch Web Page Content ---
# Use curl to fetch the web page content.
# -s: Silent mode (don't show progress or error messages).
# -S: Show error messages even in silent mode.
# -L: Follow redirects.
echo "Fetching content from '$URL'..."
PAGE_CONTENT=$(curl -sSL "$URL")

# Check if curl command was successful
if [ $? -ne 0 ]; then
  echo "Error: Failed to fetch content from '$URL'. Check your internet connection or URL." >&2
  exit 1
fi

# --- Extract License from Content ---
# This section has been updated to handle the new HTML structure.
# First, flatten the HTML content into a single line. This is crucial for grep -oP
# to match across what were originally newlines in the source HTML.
# tr -d '\n': deletes all newline characters.
# tr -s ' ': squeezes multiple spaces into a single space.
echo "Extracting license..."
CLEANED_PAGE_CONTENT=$(echo "$PAGE_CONTENT" | tr -d '\n' | tr -s ' ')

# Now, use grep with Perl-compatible regular expressions (-P) to find the license.
# The regex targets the <a> tag with `data-test-id="UnitHeader-license"` and
# captures the text content within that tag.
# '<a\s+[^>]*data-test-id="UnitHeader-license"[^>]*>\s*\K(.*?)(?=\s*</a>)'
#   - '<a\s+': Matches the start of an <a> tag followed by one or more whitespace characters.
#   - '[^>]*': Matches any character except '>' zero or more times (for attributes).
#   - 'data-test-id="UnitHeader-license"': Matches the specific data attribute.
#   - '[^>]*>': Matches any other attributes until the closing '>' of the <a> tag.
#   - '\s*\K': Matches zero or more whitespace characters, then resets the starting point
#              of the match (`\K`) so that what precedes it is not included in the output.
#   - '(.*?)': Captures any characters non-greedily (the actual license name).
#   - '(?=\s*</a>)': A positive lookahead assertion. It ensures that the captured
#                    license name is followed by zero or more whitespace characters and the
#                    closing </a> tag, but does not include these in the match.
FOUND_LICENSE=$(echo "$CLEANED_PAGE_CONTENT" | grep -oP '<a\s+[^>]*data-test-id="UnitHeader-license"[^>]*>\s*\K(.*?)(?=\s*</a>)' | head -n 1 | sed 's/^\s*//;s/\s*$//')

# Check if a license was found
if [ -z "$FOUND_LICENSE" ]; then
  echo "Error: Could not find license information on the page '$URL' with the updated regex." >&2
  exit 1
fi

echo "Retrieved license: $FOUND_LICENSE"

# --- Update File ---
# Use sed to perform an in-place replacement.
# The 's/pattern/replacement/' command performs the substitution.
# 'license: TODO' is the literal string to be replaced.
# 'license: ${FOUND_LICENSE}' is the replacement string, which includes the dynamically found license.
# Double quotes around the sed pattern allow shell variable expansion for ${FOUND_LICENSE}.
# The '-i' flag enables in-place editing.
# Note for macOS users: On some systems (like macOS), 'sed -i' requires a backup extension,
# e.g., 'sed -i ".bak"'. For broader compatibility, this script assumes GNU sed behavior.
# If you are on macOS and it fails, try 'sed -i ""' or 'sed -i.bak'.
echo "Updating license in '$FILE'..."
sed -i "s/license: TODO/license: ${FOUND_LICENSE}/" "$FILE"

echo "License information in '$FILE' successfully updated to: license: $FOUND_LICENSE"
