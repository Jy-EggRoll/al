set -gx ALIAS_SHELL 'fish'

function al
  switch "$argv[1]"
    case add rm
      command al $argv
      set -l __al_ret $status
      if test $__al_ret -eq 0
        if test "$argv[1]" = rm
          functions -e $argv[2] 2>/dev/null
        end
        source (command al init fish | psub)
      end
      return $__al_ret
    case '*'
      command al $argv
  end
end
