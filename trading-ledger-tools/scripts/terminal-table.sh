#!/usr/bin/env bash

draw_table() {
  local right_columns="${1:-}"
  local aligned

  if [[ -n "$right_columns" ]]; then
    aligned="$(column -t -s '|' -o '│' -R "$right_columns")"
  else
    aligned="$(column -t -s '|' -o '│')"
  fi

  [[ -z "$aligned" ]] && return

  local -a widths=()
  local -a cells=()
  local line cell_width
  local i

  while IFS= read -r line; do
    IFS='│' read -r -a cells <<< "$line"
    for ((i = 0; i < ${#cells[@]}; i++)); do
      cell_width="$(printf '%s' "${cells[$i]}" | wc -L)"
      if [[ -z "${widths[$i]+set}" ]] || ((cell_width > widths[i])); then
        widths[$i]="$cell_width"
      fi
    done
  done <<< "$aligned"

  local top='┌'
  local divider='├'
  local bottom='└'
  local segment

  for ((i = 0; i < ${#widths[@]}; i++)); do
    printf -v segment '%*s' "$((widths[i] + 2))" ''
    segment="${segment// /─}"
    top+="$segment"
    divider+="$segment"
    bottom+="$segment"
    if ((i + 1 < ${#widths[@]})); then
      top+='┬'
      divider+='┼'
      bottom+='┴'
    fi
  done
  top+='┐'
  divider+='┤'
  bottom+='┘'

  printf '%s\n' "$top"

  local row_number=0
  local padding
  while IFS= read -r line; do
    IFS='│' read -r -a cells <<< "$line"
    printf '│'
    for ((i = 0; i < ${#widths[@]}; i++)); do
      cell_width="$(printf '%s' "${cells[$i]:-}" | wc -L)"
      padding=$((widths[i] - cell_width))
      printf ' %s%*s │' "${cells[$i]:-}" "$padding" ''
    done
    printf '\n'
    if ((row_number == 0)); then
      printf '%s\n' "$divider"
    fi
    row_number=$((row_number + 1))
  done <<< "$aligned"

  printf '%s\n' "$bottom"
}
