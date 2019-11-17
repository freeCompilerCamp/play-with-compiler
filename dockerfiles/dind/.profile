export PS1='\[\033[01;32m\]\u@\h:\W$ \[\033[0m\]'

export PATH=$HOME/bin:$HOME/.local/bin:$PATH
clear
cat /etc/motd
#echo $BASHPID > /var/run/cwd

alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias ls='ls --color=auto'
alias vi='vim'
