
syntax on
set autoindent
set number
set printoptions=number:y
set encoding=utf-8
set wrap
set shiftwidth=4
set showmode
set warn
set tabstop=4
set expandtab
set stal=1
set wrapscan
set dir=~
set backupdir=~
set autochdir
set nospell
set ruler
set paste
set cole=0
if has("autocmd")
    au FileType html,css setlocal shiftwidth=2 tabstop=2
    au BufRead,BufNewFile *.md set filetype=markdown
    au BufReadPost * if line("'\"") > 0 && line("'\"") <= line("$")
        \| exe "normal! g'\"" | endif
endif
