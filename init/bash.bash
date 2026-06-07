export ALIAS_SHELL="bash"

al() {
  case "$1" in
    add|rm)
      command al "$@"
      local __al_ret=$?
      if [ "$__al_ret" -eq 0 ]; then
        if [ "$1" = "rm" ]; then
          unalias "${@:2}" 2>/dev/null
        fi
        eval "$(command al init bash)"
      fi
      return $__al_ret
      ;;
    *)
      command al "$@"
      ;;
  esac
}
