module github.com/f-secure-foundry/GoKey

go 1.15

require (
	github.com/f-secure-foundry/armoryctl v0.0.0-20201119100859-4cc530b6bd63
	github.com/f-secure-foundry/tamago v0.0.0-20201127101424-e0d521d1ec51
	github.com/gsora/fidati v0.0.0-20201124162830-9f1dccf32431
	github.com/hsanjuan/go-nfctype4 v0.0.1
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4
	golang.org/x/crypto v0.0.0-20201124201722-c8d3bf9c5392
	golang.org/x/sys v0.0.0-20201126233918-771906719818 // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gvisor.dev/gvisor v0.0.0-20201126084313-ad83112423ec
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20200812210008-801bb984d4b1
