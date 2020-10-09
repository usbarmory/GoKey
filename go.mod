module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/f-secure-foundry/tamago v0.0.0-20201009081246-b13354ba7d54
	github.com/hsanjuan/go-nfctype4 v0.0.1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	golang.org/x/sys v0.0.0-20201009025420-dfb3f7c4e634 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gvisor.dev/gvisor v0.0.0-20201009003428-07b1d7413e8b
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20200812210008-801bb984d4b1
