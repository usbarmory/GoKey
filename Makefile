# https://github.com/f-secure-foundry/GoKey
#
# Copyright (c) F-Secure Corporation
# https://foundry.f-secure.com
#
# Use of this source code is governed by the license
# that can be found in the LICENSE file.

BUILD_USER = $(shell whoami)
BUILD_HOST = $(shell hostname)
BUILD_DATE = $(shell /bin/date -u "+%Y-%m-%d %H:%M:%S")
BUILD_TAGS = "linkramsize"
BUILD = ${BUILD_USER}@${BUILD_HOST} on ${BUILD_DATE}
REV = $(shell git rev-parse --short HEAD 2> /dev/null)
PKG = "github.com/f-secure-foundry/GoKey"

APP := gokey
GOENV := GO_EXTLINK_ENABLED=0 CGO_ENABLED=0 GOOS=tamago GOARM=7 GOARCH=arm
TEXT_START := 0x80010000 # ramStart (defined in imx6/imx6ul/memory.go) + 0x10000
GOFLAGS := -tags ${BUILD_TAGS} -ldflags "-s -w -T $(TEXT_START) -E _rt0_arm_tamago -R 0x1000 -X '${PKG}/internal.Build=${BUILD}' -X '${PKG}/internal.Revision=${REV}'"
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
		echo 'You need to set the TAMAGO variable to a compiled version of https://github.com/f-secure-foundry/tamago-go'; \
		exit 1; \
	fi

check_usbarmory_git:
	@if [ "${USBARMORY_GIT}" == "" ]; then \
		echo 'You need to set the USBARMORY_GIT variable to the path of a clone of'; \
		echo '  https://github.com/f-secure-foundry/usbarmory'; \
		exit 1; \
	fi

check_hab_keys:
	@if [ "${HAB_KEYS}" == "" ]; then \
		echo 'You need to set the HAB_KEYS variable to the path of secure boot keys'; \
		echo 'See https://github.com/f-secure-foundry/usbarmory/wiki/Secure-boot-(Mk-II)'; \
		exit 1; \
	fi

dcd:
	echo $(GOMODCACHE)
	echo $(TAMAGO_PKG)
	cp -f $(GOMODCACHE)/$(TAMAGO_PKG)/board/f-secure/usbarmory/mark-two/imximage.cfg $(APP).dcd

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
	cd $(CURDIR) && ${TAMAGO} generate
	go build -tags vpcd -o $(CURDIR)/gokey_vpcd || (rm -f $(CURDIR)/tmp.go && exit 1)
	rm -f $(CURDIR)/tmp.go

#### dependencies ####

$(APP): check_tamago check_bundled_keys
	cd $(CURDIR) && ${TAMAGO} generate && \
	$(GOENV) $(TAMAGO) build $(GOFLAGS) -o $(CURDIR)/$(APP) || (rm -f $(CURDIR)/tmp.go && exit 1)
	rm -f $(CURDIR)/tmp.go

$(APP).dcd: check_tamago
$(APP).dcd: GOMODCACHE=$(shell ${TAMAGO} env GOMODCACHE)
$(APP).dcd: TAMAGO_PKG=$(shell grep "github.com/f-secure-foundry/tamago " go.mod | awk '{print $$1"@"$$2}')
$(APP).dcd: dcd

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

$(APP)-signed.imx: check_usbarmory_git check_hab_keys $(APP).imx
	${USBARMORY_GIT}/software/secure_boot/usbarmory_csftool \
		--csf_key ${HAB_KEYS}/CSF_1_key.pem \
		--csf_crt ${HAB_KEYS}/CSF_1_crt.pem \
		--img_key ${HAB_KEYS}/IMG_1_key.pem \
		--img_crt ${HAB_KEYS}/IMG_1_crt.pem \
		--table   ${HAB_KEYS}/SRK_1_2_3_4_table.bin \
		--index   1 \
		--image   $(APP).imx \
		--output  $(APP).csf && \
	cat $(APP).imx $(APP).csf > $(APP)-signed.imx
