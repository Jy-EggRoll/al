set -gx ALIAS_SHELL 'fish'

function al
  switch "$argv[1]"
    case add rm
      command al $argv
      set -l __al_ret $status
      if test $__al_ret -eq 0
        source (command al init fish | psub)
      end
      return $__al_ret
    case '*'
      command al $argv
  end
end
