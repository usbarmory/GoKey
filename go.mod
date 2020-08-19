module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/f-secure-foundry/tamago v0.0.0-20200819202753-3cc870d9c549
	github.com/hsanjuan/go-nfctype4 v0.0.0-20200427083210-bd42ed8410a1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20200728195943-123391ffb6de
	golang.org/x/sys v0.0.0-20200819171115-d785dc25833f // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gvisor.dev/gvisor v0.0.0-20200819190334-5cf330106a01
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20200812210008-801bb984d4b1
