module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/f-secure-foundry/armoryctl v0.0.0-20201111102916-d2bb1b09dba7
	github.com/f-secure-foundry/tamago v0.0.0-20201113223423-3d056ea6004a
	github.com/gsora/fidati v0.0.0-20201117203042-f53e0d9ed50e
	github.com/hsanjuan/go-nfctype4 v0.0.1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20201117144127-c1f2f97bffc9
	golang.org/x/sys v0.0.0-20201117170446-d9b008d0a637 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gvisor.dev/gvisor v0.0.0-20201117183629-05d2a26f7a86
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20200812210008-801bb984d4b1
