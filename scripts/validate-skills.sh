#!/usr/bin/env bash
# DevFlow Skills Validation Script
# Run locally or in CI to validate skill name consistency.
# Usage: ./scripts/validate-skills.sh [--ci]
set -euo pipefail

CI_MODE=false
[ "${1:-}" = "--ci" ] && CI_MODE=true

errors=0

red()   { echo -e "\033[31m$*\033[0m"; }
green() { echo -e "\033[32m$*\033[0m"; }

ok() {
  [ "$errors" -eq 0 ] && green "OK: $*"
}

# 1. skill-bindings.json exists
echo "=== 1. skill-bindings.json ==="
if [ ! -f skill-bindings.json ]; then
  red "ERROR: skill-bindings.json not found"
  errors=$((errors + 1))
else
  green "OK: found"
fi

# 2. Frontmatter name == directory name
echo ""
echo "=== 2. Frontmatter name == directory name ==="
while IFS= read -r -d '' f; do
  dir="$(basename "$(dirname "$f")")"
  fm="$(head -10 "$f" | grep "^name:" | head -1 | sed 's/^name: *//' | tr -d '"'"'" | xargs)"
  if [ -z "$fm" ]; then
    red "ERROR: $f — missing 'name' in frontmatter"
    errors=$((errors + 1))
  elif [ "$fm" != "$dir" ]; then
    red "ERROR: $f — name '$fm' != dir '$dir'"
    errors=$((errors + 1))
  fi
done < <(find plugins -name 'SKILL.md' -not -path '*/.git/*' -print0)
ok "all frontmatter names match"

# 3. Parse bindings
echo ""
echo "=== 3. Parse skill-bindings.json ==="
if [ ! -f skill-bindings.json ]; then
  red "ERROR: cannot parse"
  errors=$((errors + 1))
else
  TMP_NAMES="$(mktemp)"
  python3 -c "
import json
with open('skill-bindings.json') as f:
    data = json.load(f)
for plugin, skills in data.get('allSkills', {}).items():
    for s in skills:
        print(s)
" > "$TMP_NAMES"

  # 4. Every SKILL.md in allSkills
  echo ""
  echo "=== 4. Every SKILL.md in allSkills ==="
  while IFS= read -r -d '' f; do
    dir="$(basename "$(dirname "$f")")"
    if ! grep -qxF "$dir" "$TMP_NAMES"; then
      red "ERROR: $f — skill '$dir' not in skill-bindings.json allSkills"
      errors=$((errors + 1))
    fi
  done < <(find plugins -name 'SKILL.md' -not -path '*/.git/*' -print0)
  ok "all SKILL.md files declared"

  # 5. Every allSkills entry has file
  echo ""
  echo "=== 5. Every allSkills entry has file ==="
  while IFS= read -r name; do
    found="$(find plugins -path "*/${name}/SKILL.md" -not -path '*/.git/*' -print -quit)"
    if [ -z "$found" ]; then
      red "ERROR: allSkills '$name' — no SKILL.md found"
      errors=$((errors + 1))
    fi
  done < "$TMP_NAMES"
  ok "all allSkills have files"

  # 6. Plugin command cross-reference
  echo ""
  echo "=== 6. Plugin command binding cross-ref ==="
  TMP_ERR="$(mktemp)"
  python3 -c "
import json, sys
with open('skill-bindings.json') as f:
    data = json.load(f)
all_skills = data.get('allSkills', {})
errs = 0
for plugin, plugin_data in data.get('plugins', {}).items():
    commands = plugin_data.get('commands', {})
    for cmd, cmd_data in commands.items():
        primary = cmd_data.get('primarySkill', '')
        if primary and primary not in all_skills.get(plugin, []):
            print(f'ERROR: plugin/{plugin} command/{cmd} — primarySkill \"{primary}\" not in allSkills.{plugin}')
            errs += 1
        for extra in cmd_data.get('additionalSkills', []):
            if extra not in all_skills.get(plugin, []):
                print(f'ERROR: plugin/{plugin} command/{cmd} — additionalSkill \"{extra}\" not in allSkills.{plugin}')
                errs += 1
sys.exit(errs)
" > "$TMP_ERR" 2>&1 || true
  while IFS= read -r line; do
    red "$line"
    errors=$((errors + 1))
  done < "$TMP_ERR"
  ok "plugin command bindings consistent"

  rm -f "$TMP_NAMES" "$TMP_ERR"
fi

# Summary
echo ""
echo "============================================"
if [ "$errors" -gt 0 ]; then
  red "FAILED: $errors error(s)"
  $CI_MODE && exit 1
else
  green "SUCCESS: all validations passed"
fi
