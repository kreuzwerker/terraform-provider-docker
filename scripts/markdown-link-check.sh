#!/usr/bin/env bash

# Local script runner for recursive markdown-link-check

echo "==> Checking Markdown links..."

error_file="markdown-link-check-errors.txt"
output_file="markdown-link-check-output.txt"

rm -f "$error_file" "$output_file"

docker run --rm -i -t \
  -v "$(pwd)":/github/workspace:ro \
  -w /github/workspace \
  --entrypoint /usr/bin/find \
  ghcr.io/tcort/markdown-link-check:stable \
  docs -type f -name "*.md" -exec /src/markdown-link-check --config .markdownlinkcheck.json --quiet --verbose {} \; \
  | tee -a "${output_file}"

docker run --rm -i -t \
  -v "$(pwd)":/github/workspace:ro \
  -w /github/workspace \
  --entrypoint /usr/bin/find \
  ghcr.io/tcort/markdown-link-check:stable \
  website \( -type f -name "*.md" -or -name "*.markdown" \) -exec /src/markdown-link-check --config .markdownlinkcheck.json --quiet --verbose {} \; \
  | tee -a "${output_file}"

touch "${error_file}"
PREVIOUS_LINE=""
while IFS= read -r LINE; do
  if [[ $LINE = *"FILE"* ]]; then
    PREVIOUS_LINE=$LINE
    if [[ $(tail -1 "${error_file}") != *FILE* ]]; then
        echo -e "\n" >> "${error_file}"
        echo "$LINE" >> "${error_file}"
    fi
  elif [[ $LINE = *"✖"* ]] && [[ $PREVIOUS_LINE = *"FILE"* ]]; then
    echo "$LINE" >> "${error_file}"
  else 
    PREVIOUS_LINE=""
  fi
done < "${output_file}"

if grep -q "ERROR:" "${output_file}"; then
  echo -e "==================> MARKDOWN LINK CHECK FAILED <=================="
  if [[ $(tail -1 "${error_file}") = *FILE* ]]; then
    sed '$d' "${error_file}"
  else
    cat "${error_file}"
  fi
  printf "\n"
  echo -e "=================================================================="
  exit 1
else
  echo -e "==================> MARKDOWN LINK CHECK SUCCESS <=================="
  printf "\n"
  echo -e "[✔] All links are good!"
  printf "\n"
  echo -e "==================================================================="
fi

rm -f "$error_file" "$output_file"

exit 0