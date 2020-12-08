module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/f-secure-foundry/armoryctl v0.0.0-20201119100859-4cc530b6bd63
	github.com/f-secure-foundry/tamago v0.0.0-20201207220108-ec548628f80d
	github.com/gsora/fidati v0.0.0-20201207184003-5d0268ed7528
	github.com/hsanjuan/go-nfctype4 v0.0.1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20201203163018-be400aefbc4c
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20201207223542-d4d67f95c62d // indirect
	golang.org/x/term v0.0.0-20201207232118-ee85cb95a76b // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gvisor.dev/gvisor v0.0.0-20201208020054-9c198e5df421
	periph.io/x/periph v3.6.5+incompatible // indirect
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20200812210008-801bb984d4b1
