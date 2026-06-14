#!/usr/bin/env bash

# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# Usage: ./hack/sync-sig-node-with-project.sh [--update-keps] <project-url>
# Example: ./hack/sync-sig-node-with-project.sh https://github.com/orgs/kubernetes/projects/265
# Example: ./hack/sync-sig-node-with-project.sh --update-keps https://github.com/orgs/kubernetes/projects/265
#
# For each issue in the GitHub project, finds the corresponding kep.yaml under
# keps/sig-node/ by issue number prefix and compares Approvers and Reviewers
# between the project and the KEP file.
#
# With --update-keps, for each non-empty project field, adds missing handles
# to the kep.yaml approvers/reviewers lists with the appropriate annotation.
#
# Requires: gh CLI (with read:project scope), jq

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KEPS_DIR="${REPO_ROOT}/keps/sig-node"

UPDATE_KEPS=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --update-keps)
      UPDATE_KEPS=true
      shift
      ;;
    -*)
      echo "Unknown flag: $1" >&2
      exit 1
      ;;
    *)
      PROJECT_URL="$1"
      shift
      ;;
  esac
done

if [[ -z "${PROJECT_URL:-}" ]]; then
  echo "Usage: $0 [--update-keps] <project-url>" >&2
  echo "Example: $0 https://github.com/orgs/kubernetes/projects/265" >&2
  exit 1
fi

# Check dependencies
for cmd in gh jq; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "Error: '$cmd' is required but not found in PATH" >&2
    exit 1
  fi
done

# Parse org and project number from URL
if [[ "$PROJECT_URL" =~ github\.com/orgs/([^/]+)/projects/([0-9]+) ]]; then
  ORG="${BASH_REMATCH[1]}"
  PROJECT_NUMBER="${BASH_REMATCH[2]}"
else
  echo "Error: URL must match https://github.com/orgs/<org>/projects/<number>" >&2
  exit 1
fi

echo "Fetching project #${PROJECT_NUMBER} for org '${ORG}'..."

# Step 1: Get project metadata and field definitions
PROJECT_DATA=$(gh api graphql -f query='
  query($org: String!, $number: Int!) {
    organization(login: $org) {
      projectV2(number: $number) {
        id
        title
        fields(first: 50) {
          nodes {
            ... on ProjectV2Field {
              id
              name
            }
            ... on ProjectV2SingleSelectField {
              id
              name
            }
            ... on ProjectV2IterationField {
              id
              name
            }
          }
        }
      }
    }
  }
' -f org="$ORG" -F number="$PROJECT_NUMBER")

PROJECT_TITLE=$(echo "$PROJECT_DATA" | jq -r '.data.organization.projectV2.title')
echo "Project: ${PROJECT_TITLE}"
echo ""

# Normalize a handle: lowercase, strip leading @, trim whitespace
normalize_handle() {
  echo "$1" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed 's/^@//' | tr '[:upper:]' '[:lower:]'
}

# Parse a comma/space-separated string of handles into sorted, one-per-line normalized list
parse_project_handles() {
  local raw="$1"
  if [[ -z "$raw" ]]; then
    return
  fi
  # Split on commas, spaces, or newlines; normalize each
  echo "$raw" | tr ',\n' ' ' | tr -s ' ' '\n' | while read -r handle; do
    [[ -z "$handle" ]] && continue
    normalize_handle "$handle"
  done | sort -u
}

# Extract handles annotated with a specific sig-node-assigned comment from kep.yaml
# Only returns handles marked with "# sig-node-assigned-approver" or "# sig-node-assigned-reviewer"
# Usage: parse_kep_handles <kep.yaml path> <approver|reviewer>
parse_kep_handles() {
  local kep_file="$1"
  local role="$2"
  { grep "# sig-node-assigned-${role}" "$kep_file" 2>/dev/null || true; } \
    | sed 's/#.*$//' \
    | sed 's/^[[:space:]]*-[[:space:]]*//' \
    | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' \
    | sed 's/^"//;s/"$//' \
    | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' \
    | while read -r handle; do
        [[ -z "$handle" ]] && continue
        normalize_handle "$handle"
      done | sort -u
}

# Check if a handle (normalized, no @) exists anywhere in a kep.yaml section (approvers or reviewers)
# Returns 0 if found, 1 if not
kep_has_handle() {
  local kep_file="$1"
  local section="$2"  # "approvers" or "reviewers"
  local handle="$3"   # normalized, no @

  # Extract all handles from the section (with or without annotations)
  local in_section=false
  while IFS= read -r line; do
    if [[ "$line" =~ ^${section}: ]]; then
      in_section=true
      continue
    fi
    if [[ "$in_section" == "true" ]]; then
      # Stop at next top-level key or blank line
      if [[ "$line" =~ ^[a-zA-Z] ]] || [[ -z "$line" ]]; then
        break
      fi
      # Extract handle from line like:   - "@SomeUser" # comment
      local extracted
      extracted=$(echo "$line" | sed 's/#.*$//' | sed 's/^[[:space:]]*-[[:space:]]*//' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed "s/^[\"']//;s/[\"']$//" | sed 's/^@//' | tr '[:upper:]' '[:lower:]')
      if [[ "$extracted" == "$handle" ]]; then
        return 0
      fi
    fi
  done < "$kep_file"
  return 1
}

# Add a handle to a kep.yaml section with sig-node-assigned annotation
# Appends as last entry in the approvers or reviewers list
add_handle_to_kep() {
  local kep_file="$1"
  local section="$2"  # "approvers" or "reviewers"
  local handle="$3"   # normalized, no @
  local role="$4"     # "approver" or "reviewer"

  local new_line="  - \"@${handle}\" # sig-node-assigned-${role}"

  # Find the last line of the section and insert after it
  local last_line_num=0
  local in_section=false
  local line_num=0
  while IFS= read -r line; do
    line_num=$((line_num + 1))
    if [[ "$line" =~ ^${section}: ]]; then
      in_section=true
      last_line_num=$line_num
      continue
    fi
    if [[ "$in_section" == "true" ]]; then
      if [[ "$line" =~ ^[[:space:]]*-[[:space:]] ]]; then
        last_line_num=$line_num
      else
        break
      fi
    fi
  done < "$kep_file"

  if [[ "$last_line_num" -eq 0 ]]; then
    echo "    Warning: could not find '${section}:' section in ${kep_file}" >&2
    return 1
  fi

  # Insert after last_line_num using sed
  sed -i '' "${last_line_num}a\\
${new_line}
" "$kep_file"
}

# Find KEP directory for a given issue number
find_kep_dir() {
  local issue_number="$1"
  local match
  match=$(find "$KEPS_DIR" -maxdepth 1 -type d -name "${issue_number}-*" 2>/dev/null | head -1)
  echo "$match"
}

# Compare two sorted lists and report differences
# Usage: compare_lists <label> <project_list> <kep_list>
compare_lists() {
  local label="$1"
  local project_list="$2"
  local kep_list="$3"

  local only_in_project only_in_kep
  only_in_project=$(comm -23 <(echo "$project_list") <(echo "$kep_list"))
  only_in_kep=$(comm -13 <(echo "$project_list") <(echo "$kep_list"))

  if [[ -n "$only_in_project" || -n "$only_in_kep" ]]; then
    echo "  ${label} MISMATCH:"
    if [[ -n "$only_in_project" ]]; then
      echo "    In project but not in kep.yaml: $(echo "$only_in_project" | tr '\n' ' ')"
    fi
    if [[ -n "$only_in_kep" ]]; then
      echo "    In kep.yaml but not in project: $(echo "$only_in_kep" | tr '\n' ' ')"
    fi
    return 1
  fi
  return 0
}

# Step 2: Fetch all project items with pagination
CURSOR=""
HAS_NEXT="true"
ISSUES_TOTAL=0
ISSUES_FOUND=0
ISSUES_MISSING=0
ISSUES_MATCH=0
ISSUES_MISMATCH=0

while [[ "$HAS_NEXT" == "true" ]]; do
  AFTER_ARGS=()
  if [[ -n "$CURSOR" ]]; then
    AFTER_ARGS=(-f after="$CURSOR")
  fi

  ITEMS_DATA=$(gh api graphql -f query='
    query($org: String!, $number: Int!, $after: String) {
      organization(login: $org) {
        projectV2(number: $number) {
          items(first: 100, after: $after) {
            pageInfo {
              hasNextPage
              endCursor
            }
            nodes {
              id
              fieldValues(first: 50) {
                nodes {
                  ... on ProjectV2ItemFieldTextValue {
                    text
                    field {
                      ... on ProjectV2Field {
                        id
                        name
                      }
                    }
                  }
                  ... on ProjectV2ItemFieldSingleSelectValue {
                    name
                    field {
                      ... on ProjectV2SingleSelectField {
                        id
                        name
                      }
                    }
                  }
                }
              }
              content {
                ... on Issue {
                  title
                  number
                  url
                }
              }
            }
          }
        }
      }
    }
  ' -f org="$ORG" -F number="$PROJECT_NUMBER" "${AFTER_ARGS[@]}")

  HAS_NEXT=$(echo "$ITEMS_DATA" | jq -r '.data.organization.projectV2.items.pageInfo.hasNextPage')
  CURSOR=$(echo "$ITEMS_DATA" | jq -r '.data.organization.projectV2.items.pageInfo.endCursor')

  # Extract items as JSON lines
  ITEMS=$(echo "$ITEMS_DATA" | jq -c '
    .data.organization.projectV2.items.nodes[] |
    select(.content != null and .content.number != null) |
    {
      number: .content.number,
      title: .content.title,
      url: .content.url,
      approvers: ([.fieldValues.nodes[] | select(.field.name == "Approvers" or .field.name == "approvers")] | first // {text: "", name: ""} | (.text // .name // "")),
      reviewers: ([.fieldValues.nodes[] | select(.field.name == "KEPReviewers" or .field.name == "Reviewers")] | first // {text: "", name: ""} | (.text // .name // ""))
    }
  ')

  while IFS= read -r item; do
    [[ -z "$item" ]] && continue
    ISSUES_TOTAL=$((ISSUES_TOTAL + 1))

    ISSUE_NUMBER=$(echo "$item" | jq -r '.number')
    ISSUE_TITLE=$(echo "$item" | jq -r '.title')
    PROJECT_APPROVERS_RAW=$(echo "$item" | jq -r '.approvers')
    PROJECT_REVIEWERS_RAW=$(echo "$item" | jq -r '.reviewers')

    KEP_DIR=$(find_kep_dir "$ISSUE_NUMBER")
    if [[ -z "$KEP_DIR" ]]; then
      echo "#${ISSUE_NUMBER} ${ISSUE_TITLE}"
      echo "  KEP not found: no directory matching keps/sig-node/${ISSUE_NUMBER}-* "
      echo ""
      ISSUES_MISSING=$((ISSUES_MISSING + 1))
      continue
    fi

    KEP_FILE="${KEP_DIR}/kep.yaml"
    if [[ ! -f "$KEP_FILE" ]]; then
      echo "#${ISSUE_NUMBER} ${ISSUE_TITLE}"
      echo "  KEP directory found ($(basename "$KEP_DIR")) but kep.yaml is missing"
      echo ""
      ISSUES_MISSING=$((ISSUES_MISSING + 1))
      continue
    fi

    ISSUES_FOUND=$((ISSUES_FOUND + 1))

    # Parse and normalize handles from both sources
    PROJECT_APPROVERS=$(parse_project_handles "$PROJECT_APPROVERS_RAW")
    PROJECT_REVIEWERS=$(parse_project_handles "$PROJECT_REVIEWERS_RAW")
    KEP_APPROVERS=$(parse_kep_handles "$KEP_FILE" "approver")
    KEP_REVIEWERS=$(parse_kep_handles "$KEP_FILE" "reviewer")

    if [[ "$UPDATE_KEPS" == "true" ]]; then
      UPDATES=""

      # Process approvers: skip if project field is empty
      if [[ -n "$PROJECT_APPROVERS" ]]; then
        while IFS= read -r handle; do
          [[ -z "$handle" ]] && continue
          if ! kep_has_handle "$KEP_FILE" "approvers" "$handle"; then
            add_handle_to_kep "$KEP_FILE" "approvers" "$handle" "approver"
            UPDATES="${UPDATES}  Added approver @${handle}\n"
          fi
        done <<< "$PROJECT_APPROVERS"
      fi

      # Process reviewers: skip if project field is empty
      if [[ -n "$PROJECT_REVIEWERS" ]]; then
        while IFS= read -r handle; do
          [[ -z "$handle" ]] && continue
          if ! kep_has_handle "$KEP_FILE" "reviewers" "$handle"; then
            add_handle_to_kep "$KEP_FILE" "reviewers" "$handle" "reviewer"
            UPDATES="${UPDATES}  Added reviewer @${handle}\n"
          fi
        done <<< "$PROJECT_REVIEWERS"
      fi

      if [[ -n "$UPDATES" ]]; then
        echo "#${ISSUE_NUMBER} ${ISSUE_TITLE}"
        echo "  KEP: $(basename "$KEP_DIR")/kep.yaml"
        printf "%b" "$UPDATES"
        echo ""
        ISSUES_MISMATCH=$((ISSUES_MISMATCH + 1))
      else
        ISSUES_MATCH=$((ISSUES_MATCH + 1))
      fi
    else
      HAS_DIFF=false

      APPROVER_DIFF=$(compare_lists "Approvers" "$PROJECT_APPROVERS" "$KEP_APPROVERS" 2>&1) || HAS_DIFF=true
      REVIEWER_DIFF=$(compare_lists "Reviewers" "$PROJECT_REVIEWERS" "$KEP_REVIEWERS" 2>&1) || HAS_DIFF=true

      if [[ "$HAS_DIFF" == "true" ]]; then
        echo "#${ISSUE_NUMBER} ${ISSUE_TITLE}"
        echo "  KEP: $(basename "$KEP_DIR")/kep.yaml"
        [[ -n "$APPROVER_DIFF" ]] && echo "$APPROVER_DIFF"
        [[ -n "$REVIEWER_DIFF" ]] && echo "$REVIEWER_DIFF"
        echo ""
        ISSUES_MISMATCH=$((ISSUES_MISMATCH + 1))
      else
        ISSUES_MATCH=$((ISSUES_MATCH + 1))
      fi
    fi
  done <<< "$ITEMS"
done

echo "========================================="
echo "Summary:"
echo "  Total issues in project: ${ISSUES_TOTAL}"
echo "  KEP found:               ${ISSUES_FOUND}"
echo "  KEP not found:           ${ISSUES_MISSING}"
echo "  Matching:                ${ISSUES_MATCH}"
echo "  Mismatched:              ${ISSUES_MISMATCH}"
