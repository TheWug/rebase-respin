# rebase-respin

A tool to quickly edit git rebase todo files.

## Features

- Vanilla Go.
- Supports modern versions of git.
- Designed for automation.

### Compiling

Easy as can be:

```
git checkout [this repository]
cd rebase-respin
go build . && go test .
```

### Usage Overview

This tool is designed to automatically process a git rebase-todo file. You can pass it
a list of abbreviated hashes, and what to do with them, and it will read and re-write
the rebase-todo file according to your wishes.

Here's a short list of things you can do with it, which are a PITA otherwise:
* mark all commits containing a given phrase in their commit message for rewording
`git log --grep "my phrase" --pretty="filter:reword %h" | rebase-respin rebase-todo`
* run a script after every picked commit
`echo "exec default ./myscript.sh" | rebase-respin rebase-todo`
* squash an entire region into one commit
`echo "squash default\npick [first-commit]" | rebase-respin rebase-todo`
* apply any fixups which match a filter
`git log --grep "fixup! " --pretty="format:fixup %h" only/this/directory | rebase-respin rebase-todo`
* combine these tools into a fully automatic history filtering mechanism
* take over the world?

`rebase-respin --help` documents the nitty gritty.

### Star features

1. its handling of fixup and squash commits is smarter than the average rebase. If you've ever tried to autosquash when duplicate commit messages were involved you will know what I mean.  It also tries really hard to do the right thing when fixups are applied before it gets its hands on the rebase-todo file, including pulling previously applied fixups along if the commit they apply to gets moved around.
2. you can specify default actions by using the special pseudo-hash "default".
3. you can specify breaks and execs multiple times, both on specific commits and on "default", and they will apply in a deterministic order.

### Why did I write this?

I just spent literally days chewing through a 500+ commit rebase. It was extremely annoying to have to have to edit the todo file and manually search in a text editor for commit hashes so I could change the desired strategy. Total waste of time. With this tool I can now abuse git log into generating those instructions for me and apply them instantly to the todo file, saving an enormous amount of time and grunt work.
