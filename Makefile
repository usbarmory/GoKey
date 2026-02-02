# https://github.com/usbarmory/GoKey
#
# Copyright (c) The GoKey authors. All Rights Reserved.
#
# Use of this source code is governed by the license
# that can be found in the LICENSE file.

BUILD_TAGS = "linkramsize,linkprintk"
REV = $(shell git rev-parse --short HEAD 2> /dev/null)

APP := gokey
GOENV := GO_EXTLINK_ENABLED=0 CGO_ENABLED=0 GOOS=tamago GOARM=7 GOARCH=arm
TEXT_START := 0x80010000 # ramStart (defined in imx6/imx6ul/memory.go) + 0x10000
GOFLAGS := -tags ${BUILD_TAGS} -trimpath -ldflags "-s -w -T $(TEXT_START) -E _rt0_arm_tamago -R 0x1000"
SHELL = /bin/bash

.PHONY: clean

#### primary targets ####

all: $(APP)

imx: $(APP).imx

imx_signed: $(APP)-signed.imx

elf: $(APP)

#### utilities ####

check_tamago:
	@if [ "${TAMAGO}" == "" ] || [ ! -f "${TAMAGO}" ]; then \
		echo 'You need to set the TAMAGO variable to a compiled version of https://github.com/usbarmory/tamago-go'; \
		exit 1; \
	fi

check_hab_keys:
	@if [ "${HAB_KEYS}" == "" ]; then \
		echo 'You need to set the HAB_KEYS variable to the path of secure boot keys'; \
		echo 'See https://github.com/usbarmory/usbarmory/wiki/Secure-boot-(Mk-II)'; \
		exit 1; \
	fi

dcd:
	echo $(GOMODCACHE)
	echo $(TAMAGO_PKG)
	cp -f $(GOMODCACHE)/$(TAMAGO_PKG)/board/usbarmory/mk2/imximage.cfg $(APP).dcd

check_bundled_keys:
	@if { [ "${PGP_SECRET_KEY}" == "" ] || [ ! -f "${PGP_SECRET_KEY}" ]; } && { [ "${U2F_PRIVATE_KEY}" == "" ] || [ ! -f "${U2F_PRIVATE_KEY}" ]; } then \
		echo 'You need to set either PGP_SECRET_KEY or U2F_PRIVATE_KEY variables to a valid path'; \
		exit 1; \
	fi
	@if { [ -f "${U2F_PRIVATE_KEY}" ]; } && { [ "${U2F_PUBLIC_KEY}" == "" ] || [ ! -f "${U2F_PUBLIC_KEY}" ]; } then \
		echo 'You need to set the U2F_PUBLIC_KEY variable to a valid path'; \
		exit 1; \
	fi

clean:
	rm -f $(APP) gokey_vpcd
	@rm -fr $(APP).bin $(APP).imx $(APP)-signed.imx $(APP).csf $(APP).dcd

gokey_vpcd: check_bundled_keys
	cd $(CURDIR) && ${TAMAGO} generate && \
	go build -tags vpcd -o $(CURDIR)/gokey_vpcd || (rm -f $(CURDIR)/tmp.go && exit 1)
	rm -f $(CURDIR)/tmp.go

#### dependencies ####

$(APP): check_tamago check_bundled_keys
	cd $(CURDIR) && ${TAMAGO} generate && \
	$(GOENV) $(TAMAGO) build $(GOFLAGS) -o $(CURDIR)/$(APP) || (rm -f $(CURDIR)/tmp.go && exit 1)
	rm -f $(CURDIR)/tmp.go

$(APP).dcd: check_tamago
$(APP).dcd: GOMODCACHE=$(shell ${TAMAGO} env GOMODCACHE)
$(APP).dcd: TAMAGO_PKG=$(shell ${TAMAGO} list -m -f '{{.Path}}@{{.Version}}' github.com/usbarmory/tamago)
$(APP).dcd: dcd

$(APP).bin: CROSS_COMPILE=arm-none-eabi-
$(APP).bin: $(APP)
	$(CROSS_COMPILE)objcopy -j .text -j .rodata -j .shstrtab -j .typelink \
	    -j .itablink -j .gopclntab -j .go.buildinfo -j .noptrdata -j .data \
	    -j .bss --set-section-flags .bss=alloc,load,contents \
	    -j .noptrbss --set-section-flags .noptrbss=alloc,load,contents\
	    $(APP) -O binary $(APP).bin

$(APP).imx: $(APP).bin $(APP).dcd
	mkimage -n $(APP).dcd -T imximage -e $(TEXT_START) -d $(APP).bin $(APP).imx
	# Copy entry point from ELF file
	dd if=$(APP) of=$(APP).imx bs=1 count=4 skip=24 seek=4 conv=notrunc

#### secure boot ####

$(APP)-signed.imx: check_hab_keys $(APP).imx
	${TAMAGO} install github.com/usbarmory/crucible/cmd/habtool@latest
	$(shell ${TAMAGO} env GOPATH)/bin/habtool \
		-A ${HAB_KEYS}/CSF_1_key.pem \
		-a ${HAB_KEYS}/CSF_1_crt.pem \
		-B ${HAB_KEYS}/IMG_1_key.pem \
		-b ${HAB_KEYS}/IMG_1_crt.pem \
		-t ${HAB_KEYS}/SRK_1_2_3_4_table.bin \
		-x 1 \
		-i $(APP).imx \
		-o $(APP).csf && \
	cat $(APP).imx $(APP).csf > $(APP)-signed.imx
